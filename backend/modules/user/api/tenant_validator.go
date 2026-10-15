package api

import (
	"context"
	"sync"
	"time"

	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/errors"
	"saasbase/platform/tenant"
)

type cacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// TenantValidator wraps UserInternalServiceServer to implement tenant.Validator.
// It verifies that the authenticated user is an active member of the requested tenant.
type TenantValidator struct {
	internalSvc UserInternalServiceServer
	mu          sync.RWMutex
	cache       map[string]cacheEntry
	ttl         time.Duration
}

var _ tenant.Validator = (*TenantValidator)(nil)

// NewTenantValidator creates a TenantValidator backed by UserInternalServiceServer with a 1-minute cache.
func NewTenantValidator(internalSvc UserInternalServiceServer) *TenantValidator {
	return NewTenantValidatorWithTTL(internalSvc, 1*time.Minute)
}

// NewTenantValidatorWithTTL creates a TenantValidator with a custom cache TTL.
func NewTenantValidatorWithTTL(internalSvc UserInternalServiceServer, ttl time.Duration) *TenantValidator {
	if ttl <= 0 {
		ttl = 1 * time.Minute
	}
	return &TenantValidator{
		internalSvc: internalSvc,
		cache:       make(map[string]cacheEntry),
		ttl:         ttl,
	}
}

// SubscribeToEventBus connects cache invalidation to the event bus for membership changes.
func (v *TenantValidator) SubscribeToEventBus(bus interface{ Subscribe(func(authz.InvalidationEvent)) }) {
	if bus == nil {
		return
	}
	bus.Subscribe(func(evt authz.InvalidationEvent) {
		if evt.Type == authz.EventUserLeftTenant {
			v.Invalidate(evt.UserID, evt.TenantID)
		}
	})
}

// Validate checks whether the authenticated user in ctx has active access to tenantID.
func (v *TenantValidator) Validate(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return nil
	}

	id, ok := authn.FromContext(ctx)
	if !ok || id.UserID == "" {
		// If request is unauthenticated, cannot act inside a tenant
		return errors.Forbidden("tenant.forbidden", "authenticated user required for tenant access")
	}

	key := id.UserID + ":" + tenantID

	// Check cache
	v.mu.RLock()
	entry, found := v.cache[key]
	v.mu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		if !entry.allowed {
			return errors.Forbidden("tenant.forbidden", "forbidden: not an active member of this tenant")
		}
		return nil
	}

	// Query UserInternalService
	resp, err := v.internalSvc.CheckUserTenant(ctx, &CheckUserTenantRequest{
		UserId:   id.UserID,
		TenantId: tenantID,
	})
	if err != nil {
		return errors.Internal(err)
	}

	allowed := resp != nil && resp.Allowed

	// Store in cache
	v.mu.Lock()
	v.cache[key] = cacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(v.ttl),
	}
	v.mu.Unlock()

	if !allowed {
		return errors.Forbidden("tenant.forbidden", "forbidden: not an active member of this tenant")
	}

	return nil
}

// Invalidate clears cache entries for a user-tenant pair when membership changes.
func (v *TenantValidator) Invalidate(userID, tenantID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if userID != "" && tenantID != "" {
		delete(v.cache, userID+":"+tenantID)
		return
	}
	if userID != "" {
		prefix := userID + ":"
		for k := range v.cache {
			if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
				delete(v.cache, k)
			}
		}
	}
}
