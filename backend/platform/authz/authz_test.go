package authz_test

import (
	"context"
	"testing"
	"time"

	"saasbase/platform/authn"
	"saasbase/platform/authz"
	"saasbase/platform/tenant"
)

type mockPermSource struct {
	perms map[string][]authz.Permission // key: userID + ":" + tenantID
}

func (m *mockPermSource) GetUserPermissions(ctx context.Context, userID, tenantID string) ([]authz.Permission, error) {
	return m.perms[userID+":"+tenantID], nil
}

func TestCachedChecker_CheckAndWildcard(t *testing.T) {
	bus := authz.NewMemoryEventBus()
	source := &mockPermSource{
		perms: map[string][]authz.Permission{
			"alice:tenant-a": {
				{Namespace: "note.note", Action: "read"},
				{Namespace: "note.note", Action: "write"},
			},
			"bob:tenant-a": {
				{Namespace: "task.task", Action: "*"}, // action wildcard
			},
			"admin:tenant-a": {
				{Namespace: "*", Action: "*"}, // super wildcard
			},
		},
	}

	checker := authz.NewCachedChecker(source, bus, 1*time.Minute)

	ctxAlice := authn.NewContext(tenant.NewContext(context.Background(), "tenant-a"), authn.Identity{UserID: "alice", TenantID: "tenant-a"})
	ctxBob := authn.NewContext(tenant.NewContext(context.Background(), "tenant-a"), authn.Identity{UserID: "bob", TenantID: "tenant-a"})
	ctxAdmin := authn.NewContext(tenant.NewContext(context.Background(), "tenant-a"), authn.Identity{UserID: "admin", TenantID: "tenant-a"})

	// Alice has note.note read and write
	if err := checker.Check(ctxAlice, "note.note", "read"); err != nil {
		t.Fatalf("expected alice to have note.note:read, got: %v", err)
	}
	if err := checker.Check(ctxAlice, "note.note", "write"); err != nil {
		t.Fatalf("expected alice to have note.note:write, got: %v", err)
	}
	// Alice does NOT have task.task:read
	if err := checker.Check(ctxAlice, "task.task", "read"); err == nil {
		t.Fatal("expected alice to NOT have task.task:read, but succeeded")
	}

	// Bob has action wildcard on task.task
	if err := checker.Check(ctxBob, "task.task", "read"); err != nil {
		t.Fatalf("expected bob to have task.task:read via wildcard, got: %v", err)
	}
	if err := checker.Check(ctxBob, "task.task", "write"); err != nil {
		t.Fatalf("expected bob to have task.task:write via wildcard, got: %v", err)
	}
	if err := checker.Check(ctxBob, "note.note", "read"); err == nil {
		t.Fatal("expected bob to NOT have note.note:read")
	}

	// Admin has *:* wildcard
	if err := checker.Check(ctxAdmin, "anything", "any_action"); err != nil {
		t.Fatalf("expected admin to have *:* permission, got: %v", err)
	}
}

func TestCachedChecker_InvalidationOnRoleChange(t *testing.T) {
	bus := authz.NewMemoryEventBus()
	source := &mockPermSource{
		perms: map[string][]authz.Permission{
			"user-1:tenant-1": {
				{Namespace: "note.note", Action: "read"},
			},
		},
	}

	checker := authz.NewCachedChecker(source, bus, 10*time.Minute)
	ctx := authn.NewContext(tenant.NewContext(context.Background(), "tenant-1"), authn.Identity{UserID: "user-1", TenantID: "tenant-1"})

	// 1. Initial check: user-1 has read, does NOT have write
	if err := checker.Check(ctx, "note.note", "read"); err != nil {
		t.Fatalf("expected read permission: %v", err)
	}
	if err := checker.Check(ctx, "note.note", "write"); err == nil {
		t.Fatal("expected write permission to be denied initially")
	}

	// 2. Grant write permission in database source (simulating role assignment / update)
	source.perms["user-1:tenant-1"] = []authz.Permission{
		{Namespace: "note.note", Action: "read"},
		{Namespace: "note.note", Action: "write"},
	}

	// 3. Before event dispatch, cache is still hit (write is still denied)
	if err := checker.Check(ctx, "note.note", "write"); err == nil {
		t.Fatal("expected cached denial before invalidation event")
	}

	// 4. Publish invalidation event (e.g. role assigned or permissions changed)
	bus.Publish(ctx, authz.InvalidationEvent{
		Type:     authz.EventRoleAssigned,
		UserID:   "user-1",
		TenantID: "tenant-1",
	})

	// 5. Cache is invalidated immediately -> next check hits DB source and succeeds!
	if err := checker.Check(ctx, "note.note", "write"); err != nil {
		t.Fatalf("expected write permission after invalidation event, got: %v", err)
	}
}

func BenchmarkCachedChecker_Hit(b *testing.B) {
	bus := authz.NewMemoryEventBus()
	source := &mockPermSource{
		perms: map[string][]authz.Permission{
			"bench-user:tenant-bench": {
				{Namespace: "note.note", Action: "read"},
				{Namespace: "note.note", Action: "write"},
			},
		},
	}

	checker := authz.NewCachedChecker(source, bus, 10*time.Minute)
	ctx := authn.NewContext(tenant.NewContext(context.Background(), "tenant-bench"), authn.Identity{UserID: "bench-user", TenantID: "tenant-bench"})

	// Prime the cache
	if err := checker.Check(ctx, "note.note", "read"); err != nil {
		b.Fatalf("failed to prime cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx, "note.note", "read")
	}
}
