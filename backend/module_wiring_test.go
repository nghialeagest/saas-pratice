package main

import (
	"testing"

	noteapi "saasbase/modules/note/api"
	taskapi "saasbase/modules/task/api"
	tenantapi "saasbase/modules/tenant/api"
	userapi "saasbase/modules/user/api"
	"saasbase/platform/data"
	"saasbase/platform/module"
)

func TestModulesRegisterAndWire(t *testing.T) {
	provider := data.NewProvider(data.NewSharedConn("postgres://unused"), nil)

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, m := range module.All() {
		names[m.Name()] = true
	}
	for _, want := range []string{"note", "task", "user", "tenant"} {
		if !names[want] {
			t.Fatalf("module %q not registered", want)
		}
	}

	if _, err := module.Resolve[noteapi.NoteServiceServer](); err != nil {
		t.Fatalf("note contract not published: %v", err)
	}
	if _, err := module.Resolve[taskapi.TaskServiceServer](); err != nil {
		t.Fatalf("task contract not published: %v", err)
	}
	if _, err := module.Resolve[userapi.UserServiceServer](); err != nil {
		t.Fatalf("user contract not published: %v", err)
	}
	if _, err := module.Resolve[userapi.RoleServiceServer](); err != nil {
		t.Fatalf("role contract not published: %v", err)
	}
	if _, err := module.Resolve[userapi.UserInternalServiceServer](); err != nil {
		t.Fatalf("user internal contract not published: %v", err)
	}
	if _, err := module.Resolve[userapi.AuthServiceServer](); err != nil {
		t.Fatalf("auth contract not published: %v", err)
	}
	if _, err := module.Resolve[tenantapi.TenantServiceServer](); err != nil {
		t.Fatalf("tenant contract not published: %v", err)
	}

	perms := module.Permissions()
	if len(perms) != 11 {
		t.Fatalf("permissions = %d, want 11 (3 note + 2 task + 4 user + 2 tenant)", len(perms))
	}

	if len(module.Tables()) != 10 {
		t.Fatalf("tables = %d, want 10 (1 note + 1 task + 7 user + 1 tenant)", len(module.Tables()))
	}
}
