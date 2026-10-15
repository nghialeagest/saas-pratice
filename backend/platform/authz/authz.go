package authz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"saasbase/platform/authn"
	"saasbase/platform/errors"
	"saasbase/platform/tenant"
)

// Permission represents a namespace-scoped action.
type Permission struct {
	Namespace string
	Action    string
}

func (p Permission) String() string {
	return p.Namespace + ":" + p.Action
}

// Checker validates whether the caller has the required permission.
type Checker interface {
	Check(ctx context.Context, namespace, action string) error
}

// PermissionSource fetches the raw granted permissions for a given user within a tenant.
type PermissionSource interface {
	GetUserPermissions(ctx context.Context, userID, tenantID string) ([]Permission, error)
}

// EventType defines invalidation triggers.
type EventType string

const (
	EventRoleAssigned     EventType = "role_assigned"
	EventRoleRevoked      EventType = "role_revoked"
	EventRolePermsChanged EventType = "role_perms_changed"
	EventUserLeftTenant   EventType = "user_left_tenant"
)

// InvalidationEvent is dispatched when permissions or roles change.
type InvalidationEvent struct {
	Type     EventType
	UserID   string
	TenantID string
	RoleID   string
}

// EventBus provides pub/sub for permission cache invalidation.
type EventBus interface {
	Publish(ctx context.Context, evt InvalidationEvent)
	Subscribe(handler func(evt InvalidationEvent))
}

type memoryEventBus struct {
	mu          sync.RWMutex
	subscribers []func(evt InvalidationEvent)
}

func NewMemoryEventBus() EventBus {
	return &memoryEventBus{}
}

func (b *memoryEventBus) Publish(_ context.Context, evt InvalidationEvent) {
	b.mu.RLock()
	subs := make([]func(evt InvalidationEvent), len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subs {
		sub(evt)
	}
}

func (b *memoryEventBus) Subscribe(handler func(evt InvalidationEvent)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, handler)
}

// CachedChecker implements high-performance in-memory cached permission checks.
// It subscribes to an EventBus to invalidate cache entries immediately when roles or permissions change.
type CachedChecker struct {
	source   PermissionSource
	eventBus EventBus
	ttl      time.Duration

	mu    sync.RWMutex
	cache map[string]cacheItem
}

type cacheItem struct {
	perms     map[string]struct{}
	expiresAt time.Time
}

// NewCachedChecker initializes a CachedChecker with an invalidation event listener.
func NewCachedChecker(source PermissionSource, bus EventBus, ttl time.Duration) *CachedChecker {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if bus == nil {
		bus = NewMemoryEventBus()
	}

	c := &CachedChecker{
		source:   source,
		eventBus: bus,
		ttl:      ttl,
		cache:    make(map[string]cacheItem),
	}

	bus.Subscribe(c.handleInvalidation)
	return c
}

func (c *CachedChecker) EventBus() EventBus {
	return c.eventBus
}

func (c *CachedChecker) handleInvalidation(evt InvalidationEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If specific user and tenant, invalidate targeted key
	if evt.UserID != "" && evt.TenantID != "" {
		delete(c.cache, c.cacheKey(evt.UserID, evt.TenantID))
		return
	}

	// If tenant-level change (e.g. role permissions updated), invalidate all entries for that tenant
	if evt.TenantID != "" {
		suffix := ":" + evt.TenantID
		for k := range c.cache {
			if strings.HasSuffix(k, suffix) {
				delete(c.cache, k)
			}
		}
		return
	}

	// User-level change across all tenants
	if evt.UserID != "" {
		prefix := evt.UserID + ":"
		for k := range c.cache {
			if strings.HasPrefix(k, prefix) {
				delete(c.cache, k)
			}
		}
		return
	}

	// Fallback: clear entire cache
	c.cache = make(map[string]cacheItem)
}

func (c *CachedChecker) cacheKey(userID, tenantID string) string {
	return userID + ":" + tenantID
}

// Check verifies that the subject in ctx has the required namespace:action permission.
func (c *CachedChecker) Check(ctx context.Context, namespace, action string) error {
	id, ok := authn.FromContext(ctx)
	if !ok || id.UserID == "" {
		return errors.Unauthorized("authentication required")
	}

	// The tenant in context is the single source of truth. Falling back to the
	// tenant inside the JWT would let a caller omit the tenant header and have
	// permissions evaluated against their own tenant while the data layer reads
	// and writes host scope.
	tenantID := tenant.FromContext(ctx)

	perms, err := c.getPermissions(ctx, id.UserID, tenantID)
	if err != nil {
		return err
	}

	if c.hasPermission(perms, namespace, action) {
		return nil
	}

	return errors.Forbidden("authz.permission_denied", fmt.Sprintf("permission denied for %s:%s", namespace, action))
}

func (c *CachedChecker) getPermissions(ctx context.Context, userID, tenantID string) (map[string]struct{}, error) {
	key := c.cacheKey(userID, tenantID)

	// 1. Fast path: Read lock
	c.mu.RLock()
	item, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Now().Before(item.expiresAt) {
		return item.perms, nil
	}

	// 2. Slow path: Fetch from DB source
	rawPerms, err := c.source.GetUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	permSet := make(map[string]struct{}, len(rawPerms))
	for _, p := range rawPerms {
		permSet[p.Namespace+":"+p.Action] = struct{}{}
	}

	// 3. Cache write lock
	c.mu.Lock()
	c.cache[key] = cacheItem{
		perms:     permSet,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return permSet, nil
}

func (c *CachedChecker) hasPermission(perms map[string]struct{}, namespace, action string) bool {
	// Full wildcard
	if _, ok := perms["*:*"]; ok {
		return true
	}
	// Action wildcard (e.g. "note.note:*")
	if _, ok := perms[namespace+":*"]; ok {
		return true
	}
	// Exact match
	if _, ok := perms[namespace+":"+action]; ok {
		return true
	}
	return false
}
