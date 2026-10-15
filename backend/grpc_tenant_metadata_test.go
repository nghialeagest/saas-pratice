package main

import (
	"context"
	"net"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	noteapi "saasbase/modules/note/api"
	"saasbase/platform/data"
	"saasbase/platform/grpcx"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

// TestGRPC_TenantMetadataPropagation tests:
// 1. Calling NoteService over gRPC with x-tenant-id: tenant-a only sees tenant A data.
// 2. Calling NoteService without x-tenant-id metadata is REJECTED (fail-closed) and does not fall back to host scope.
func TestGRPC_TenantMetadataPropagation(t *testing.T) {
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

	// 1. Setup in-memory gRPC server with TenantUnaryInterceptor and ErrorUnaryInterceptor
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpcx.ErrorUnaryInterceptor(),
		grpcx.TenantUnaryInterceptor(),
	))
	noteapi.RegisterNoteServiceServer(server, noteService)

	go func() {
		if err := server.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("grpc server error: %v", err)
		}
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	// 2. Setup gRPC client connection
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := noteapi.NewNoteServiceClient(conn)

	// Seed Note for tenant-a and Note for tenant-b
	ctxA := tenant.NewContext(context.Background(), "tenant-a")
	ctxB := tenant.NewContext(context.Background(), "tenant-b")

	noteA, err := noteService.CreateNote(ctxA, &noteapi.CreateNoteRequest{
		Title: "Note A in Tenant A",
		Body:  "Alpha content",
	})
	if err != nil {
		t.Fatalf("failed to create note A: %v", err)
	}

	noteB, err := noteService.CreateNote(ctxB, &noteapi.CreateNoteRequest{
		Title: "Note B in Tenant B",
		Body:  "Beta content",
	})
	if err != nil {
		t.Fatalf("failed to create note B: %v", err)
	}

	// 3. Test: Call gRPC with metadata x-tenant-id: tenant-a
	grpcCtxA := metadata.AppendToOutgoingContext(context.Background(), "x-tenant-id", "tenant-a")

	// 3a. Successfully reads note A
	gotA, err := client.GetNote(grpcCtxA, &noteapi.GetNoteRequest{Id: noteA.GetId()})
	if err != nil {
		t.Fatalf("failed to get note A with tenant-a metadata: %v", err)
	}
	if gotA.GetTitle() != "Note A in Tenant A" {
		t.Fatalf("unexpected title: %s", gotA.GetTitle())
	}

	// 3b. Cannot read note B with tenant-a metadata -> must be NotFound
	_, err = client.GetNote(grpcCtxA, &noteapi.GetNoteRequest{Id: noteB.GetId()})
	if err == nil {
		t.Fatalf("SECURITY VULNERABILITY: read tenant B's note using tenant-a metadata!")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound error, got %v: %v", st.Code(), err)
	}

	// 3c. ListNotes with tenant-a metadata only returns tenant A's notes
	listResp, err := client.ListNotes(grpcCtxA, &noteapi.ListNotesRequest{})
	if err != nil {
		t.Fatalf("failed to list notes for tenant-a: %v", err)
	}
	if len(listResp.GetItems()) != 1 || listResp.GetItems()[0].GetId() != noteA.GetId() {
		t.Fatalf("expected exactly note A in tenant-a list, got %d notes", len(listResp.GetItems()))
	}

	// 4. CRITICAL TEST: Missing metadata x-tenant-id MUST be rejected (fail-closed)
	// Without metadata, it MUST NOT fall back to host scope.
	emptyCtx := context.Background()
	_, err = client.GetNote(emptyCtx, &noteapi.GetNoteRequest{Id: noteA.GetId()})
	if err == nil {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY: gRPC request with NO tenant metadata succeeded!")
	}
	stEmpty, _ := status.FromError(err)
	if stEmpty.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing tenant metadata, got code %v: %v", stEmpty.Code(), err)
	}

	// Also verify ListNotes without metadata is rejected
	_, err = client.ListNotes(emptyCtx, &noteapi.ListNotesRequest{})
	if err == nil {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY: gRPC ListNotes with NO tenant metadata succeeded!")
	}
	stEmptyList, _ := status.FromError(err)
	if stEmptyList.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing tenant metadata, got code %v: %v", stEmptyList.Code(), err)
	}
}
