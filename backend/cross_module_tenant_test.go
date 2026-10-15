package main

import (
	"context"
	"os"
	"testing"

	noteapi "saasbase/modules/note/api"
	taskapi "saasbase/modules/task/api"
	"saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

// TestCrossModule_TenantContextPreserved verifies that when a service in one module (task)
// calls a service in another module (note), the tenant context is strictly preserved.
// Attempting to reference tenant B's note from tenant A's task must fail.
func TestCrossModule_TenantContextPreserved(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	allTables := module.Tables()
	_, provider := data.IsolatedTestDB(t, dsn, allTables...)

	if err := module.InitAll(module.Deps{DB: provider}); err != nil {
		t.Fatalf("failed to init modules: %v", err)
	}

	noteService, err := module.Resolve[noteapi.NoteServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve NoteServiceServer: %v", err)
	}

	taskService, err := module.Resolve[taskapi.TaskServiceServer]()
	if err != nil {
		t.Fatalf("failed to resolve TaskServiceServer: %v", err)
	}

	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	// 1. Create a note in tenant B
	noteB, err := noteService.CreateNote(ctxB, &noteapi.CreateNoteRequest{
		Title: "Tenant B Confidential Note",
		Body:  "Secret content belonging to tenant B",
	})
	if err != nil {
		t.Fatalf("failed to create note in tenant B: %v", err)
	}
	if noteB.GetId() == "" {
		t.Fatal("expected note in tenant B to have non-empty ID")
	}

	// Verify tenant B can read its note
	gotNoteB, err := noteService.GetNote(ctxB, &noteapi.GetNoteRequest{Id: noteB.GetId()})
	if err != nil || gotNoteB == nil {
		t.Fatalf("tenant B should be able to read its own note: %v", err)
	}

	// 2. CRITICAL TEST: Task creation in tenant A attempting to reference tenant B's note.
	// task.CreateTask(ctxA, {note_id: noteB.Id}) calls note.GetNote(ctxA)
	// Because ctxA has tenant-a, note.GetNote must NOT see tenant B's note.
	// Therefore, CreateTask must fail and not create the task.
	_, err = taskService.CreateTask(ctxA, &taskapi.CreateTaskRequest{
		Title:  "Malicious or Accidental Cross-Tenant Reference",
		NoteId: noteB.GetId(),
	})
	if err == nil {
		t.Fatalf("SECURITY VULNERABILITY: task.CreateTask in tenant-a succeeded with note from tenant-b!")
	}

	// 3. Verify no tasks were created in tenant A
	countResp, err := taskService.CountTasks(ctxA, &taskapi.CountTasksRequest{})
	if err != nil {
		t.Fatalf("failed to count tasks in tenant A: %v", err)
	}
	if countResp.GetCount() != 0 {
		t.Fatalf("expected 0 tasks in tenant A, found %d", countResp.GetCount())
	}

	// 4. Legitimate case: Create a note in tenant A, then create a task referencing tenant A's note
	noteA, err := noteService.CreateNote(ctxA, &noteapi.CreateNoteRequest{
		Title: "Tenant A Legitimate Note",
		Body:  "Note belonging to tenant A",
	})
	if err != nil {
		t.Fatalf("failed to create note in tenant A: %v", err)
	}

	legitTask, err := taskService.CreateTask(ctxA, &taskapi.CreateTaskRequest{
		Title:  "Tenant A Legitimate Task",
		NoteId: noteA.GetId(),
	})
	if err != nil {
		t.Fatalf("task creation within same tenant A should succeed: %v", err)
	}
	if legitTask.GetNoteTitle() != "Tenant A Legitimate Note" {
		t.Fatalf("expected note title 'Tenant A Legitimate Note', got '%s'", legitTask.GetNoteTitle())
	}
}
