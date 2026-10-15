package main

import (
	"context"
	"os"
	"testing"

	"gorm.io/gorm"

	noteapi "saasbase/modules/note/api"
	"saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

func setup(t *testing.T) (*data.Provider, noteapi.NoteServiceServer, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	provider := data.NewProvider(data.NewSharedConn(dsn), nil)
	t.Cleanup(func() { _ = provider.Close() })

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatal(err)
	}

	hostCtx := tenant.SkipFilter(tenant.NewContext(context.Background(), ""))
	db, err := provider.DB(hostCtx)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Exec("DROP TABLE IF EXISTS user_roles, user_tenants, roles, users, tasks, notes CASCADE")
	if err := db.AutoMigrate(module.Tables()...); err != nil {
		t.Fatal(err)
	}

	var svc noteapi.NoteServiceServer
	for _, m := range module.All() {
		if m.Name() == "note" {
			svc = m.(interface{ Service() noteapi.NoteServiceServer }).Service()
		}
	}
	if svc == nil {
		t.Fatal("note module not registered")
	}
	return provider, svc, db
}

func TestTenantIsolation(t *testing.T) {
	_, svc, db := setup(t)

	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	for _, title := range []string{"first", "second"} {
		if _, err := svc.CreateNote(ctxA, &noteapi.CreateNoteRequest{Title: title}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateNote(ctxB, &noteapi.CreateNoteRequest{Title: title}); err != nil {
			t.Fatal(err)
		}
	}

	countA, err := svc.CountNotes(ctxA, &noteapi.CountNotesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if countA.GetCount() != 2 {
		t.Fatalf("tenant A sees %d rows, want 2", countA.GetCount())
	}

	before, _ := svc.CountNotes(ctxB, &noteapi.CountNotesRequest{})
	if _, err := svc.CreateNote(ctxA, &noteapi.CreateNoteRequest{Title: "a only"}); err != nil {
		t.Fatal(err)
	}
	after, _ := svc.CountNotes(ctxB, &noteapi.CountNotesRequest{})
	if before.GetCount() != after.GetCount() {
		t.Fatalf("write to tenant A changed tenant B: %d -> %d", before.GetCount(), after.GetCount())
	}

	var orphans int64
	if err := db.Raw(`SELECT COUNT(*) FROM notes WHERE tenant_id IS NULL OR tenant_id = ''`).
		Scan(&orphans).Error; err != nil {
		t.Fatal(err)
	}
	if orphans != 0 {
		t.Fatalf("%d rows written without a tenant id", orphans)
	}

	hostCount, err := svc.CountNotes(tenant.NewContext(context.Background(), ""), &noteapi.CountNotesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if hostCount.GetCount() != 0 {
		t.Fatalf("host scope sees %d tenant rows, want 0", hostCount.GetCount())
	}

	all, err := svc.CountNotes(tenant.SkipFilter(context.Background()), &noteapi.CountNotesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if all.GetCount() != 5 {
		t.Fatalf("unfiltered count = %d, want 5", all.GetCount())
	}
}

func TestCrossTenantReadIsDenied(t *testing.T) {
	_, svc, _ := setup(t)

	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	created, err := svc.CreateNote(ctxA, &noteapi.CreateNoteRequest{Title: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetNote(ctxB, &noteapi.GetNoteRequest{Id: created.GetId()}); err == nil {
		t.Fatal("tenant B read a note owned by tenant A")
	}
}

func TestPublishStaysInsideTenant(t *testing.T) {
	_, svc, _ := setup(t)

	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	created, err := svc.CreateNote(ctxA, &noteapi.CreateNoteRequest{Title: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishNote(ctxB, &noteapi.PublishNoteRequest{Id: created.GetId()}); err == nil {
		t.Fatal("tenant B published a note owned by tenant A")
	}
	out, err := svc.PublishNote(ctxA, &noteapi.PublishNoteRequest{Id: created.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if out.GetStatus() != noteapi.NoteStatus_NOTE_STATUS_PUBLISHED {
		t.Fatalf("status = %v, want PUBLISHED", out.GetStatus())
	}
}
