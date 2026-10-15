package data

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"saasbase/platform/tenant"
)

// IsolatedTestDB creates a completely private, isolated PostgreSQL schema for the running test.
// This allows tests to run concurrently without table conflicts or drop/create race conditions.
func IsolatedTestDB(t *testing.T, baseDSN string, tables ...any) (*gorm.DB, *Provider) {
	t.Helper()
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	rawDB, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	// Generate clean schema name
	schemaName := fmt.Sprintf("test_schema_%s", strings.ReplaceAll(uuid.NewString(), "-", ""))

	// Create private schema
	if err := rawDB.Exec("CREATE SCHEMA " + schemaName).Error; err != nil {
		t.Fatalf("failed to create isolated schema %s: %v", schemaName, err)
	}

	// Clean up schema after test completes
	t.Cleanup(func() {
		_ = rawDB.Exec("DROP SCHEMA " + schemaName + " CASCADE")
	})

	// Format DSN with search_path set to the isolated schema
	isolatedDSN := baseDSN
	if strings.Contains(isolatedDSN, "?") {
		isolatedDSN += "&search_path=" + schemaName
	} else {
		isolatedDSN += "?search_path=" + schemaName
	}

	provider := NewProvider(NewSharedConn(isolatedDSN), nil)
	t.Cleanup(func() { _ = provider.Close() })

	testDB, err := gorm.Open(postgres.Open(isolatedDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open isolated db: %v", err)
	}

	if err := tenant.Register(testDB); err != nil {
		t.Fatalf("failed to register tenant callbacks: %v", err)
	}

	if len(tables) > 0 {
		if err := testDB.AutoMigrate(tables...); err != nil {
			t.Fatalf("failed to migrate tables in schema %s: %v", schemaName, err)
		}
	}

	return testDB, provider
}
