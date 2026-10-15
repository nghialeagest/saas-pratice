package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"saasbase/platform/data"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

func TestAllModulesTenantIsolation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	allTables := module.Tables()
	if len(allTables) == 0 {
		t.Fatal("no module tables registered")
	}

	// Use an isolated PostgreSQL schema for this test
	db, _ := data.IsolatedTestDB(t, dsn, allTables...)

	tenantTablesCount := 0

	for _, model := range allTables {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			continue
		}

		schema := stmt.Schema
		if schema == nil || schema.LookUpField("TenantID") == nil {
			// Skip models that are intentionally global (e.g., User, UserTenant)
			continue
		}

		tableName := schema.Table
		tenantTablesCount++

		t.Run(fmt.Sprintf("Table_%s", tableName), func(t *testing.T) {
			ctxA := tenant.NewContext(context.Background(), "tenant-alpha")
			ctxB := tenant.NewContext(context.Background(), "tenant-beta")
			ctxHost := tenant.NewContext(context.Background(), "")
			ctxSkip := tenant.SkipFilter(context.Background())

			// 1. Create a new instance of the model using reflection
			modelType := reflect.TypeOf(model)
			if modelType.Kind() == reflect.Ptr {
				modelType = modelType.Elem()
			}
			valA := reflect.New(modelType).Interface()

			// Set Primary Key if string field "ID" exists (raw UUID is 36 chars, fits varchar(36))
			idField := schema.LookUpField("ID")
			if idField != nil && idField.FieldType.Kind() == reflect.String {
				idField.Set(context.Background(), reflect.ValueOf(valA), uuid.NewString())
			}

			// Populate any other primary key and required fields
			if f := schema.LookUpField("UserID"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "u-"+uuid.NewString())
			}
			if f := schema.LookUpField("RoleID"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "r-"+uuid.NewString())
			}
			if f := schema.LookUpField("Title"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "title-alpha")
			}
			if f := schema.LookUpField("Name"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "name-alpha")
			}
			if f := schema.LookUpField("NormalizedName"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "NAME-ALPHA")
			}
			if f := schema.LookUpField("Status"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "ACTIVE")
			}
			if f := schema.LookUpField("Namespace"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "test.resource")
			}
			if f := schema.LookUpField("Action"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "read")
			}
			if f := schema.LookUpField("RefreshToken"); f != nil && f.FieldType.Kind() == reflect.String {
				f.Set(context.Background(), reflect.ValueOf(valA), "rt-"+uuid.NewString())
			}
			if f := schema.LookUpField("JoinedAt"); f != nil {
				f.Set(context.Background(), reflect.ValueOf(valA), time.Now())
			}
			if f := schema.LookUpField("ExpiresAt"); f != nil {
				f.Set(context.Background(), reflect.ValueOf(valA), time.Now().Add(time.Hour))
			}

			// Insert row under tenant A
			if err := db.WithContext(ctxA).Create(valA).Error; err != nil {
				t.Fatalf("failed to insert row into %s for tenant A: %v", tableName, err)
			}

			queryModel := reflect.New(modelType).Interface()

			// 2. Query as tenant B -> MUST BE 0 rows
			var countB int64
			if err := db.WithContext(ctxB).Model(queryModel).Count(&countB).Error; err != nil {
				t.Fatalf("failed to count %s for tenant B: %v", tableName, err)
			}
			if countB != 0 {
				t.Fatalf("TENANT LEAK: tenant B sees %d rows in %s, want 0", countB, tableName)
			}

			// 3. Query as host scope (empty tenant) -> MUST BE 0 rows
			var countHost int64
			if err := db.WithContext(ctxHost).Model(queryModel).Count(&countHost).Error; err != nil {
				t.Fatalf("failed to count %s for host: %v", tableName, err)
			}
			if countHost != 0 {
				t.Fatalf("TENANT LEAK: host scope sees %d tenant rows in %s, want 0", countHost, tableName)
			}

			// 4. Query as tenant A -> MUST BE >= 1 row
			var countA int64
			if err := db.WithContext(ctxA).Model(queryModel).Count(&countA).Error; err != nil {
				t.Fatalf("failed to count %s for tenant A: %v", tableName, err)
			}
			if countA == 0 {
				t.Fatalf("tenant A cannot see its own row in %s", tableName)
			}

			// 5. Query with SkipFilter -> MUST see the row
			var countSkip int64
			if err := db.WithContext(ctxSkip).Model(queryModel).Count(&countSkip).Error; err != nil {
				t.Fatalf("failed to count %s with SkipFilter: %v", tableName, err)
			}
			if countSkip == 0 {
				t.Fatalf("SkipFilter cannot see rows in %s", tableName)
			}

			// 6. RAW SQL check: Verify NO row in the table was written with empty or null tenant_id
			var orphanCount int64
			rawQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE tenant_id IS NULL OR tenant_id = ''`, tableName)
			if err := db.WithContext(ctxSkip).Raw(rawQuery).Scan(&orphanCount).Error; err != nil {
				t.Fatalf("failed to check orphans in %s: %v", tableName, err)
			}
			if orphanCount != 0 {
				t.Fatalf("DATA INTEGRITY VIOLATION: %d rows written into %s without a tenant_id", orphanCount, tableName)
			}

			// 7. Cross-tenant mutation isolation: tenant B cannot delete or update tenant A's row
			if idField != nil {
				idVal, _ := idField.ValueOf(context.Background(), reflect.ValueOf(valA))
				delResult := db.WithContext(ctxB).Model(queryModel).Where("id = ?", idVal).Delete(nil)
				if delResult.RowsAffected != 0 {
					t.Fatalf("CROSS-TENANT MUTATION: tenant B deleted %d rows belonging to tenant A in %s", delResult.RowsAffected, tableName)
				}
			}
		})
	}

	if tenantTablesCount == 0 {
		t.Fatal("no tenant-scoped tables found to test")
	}

	t.Logf("Successfully verified tenant isolation across all %d tenant-scoped tables", tenantTablesCount)
}
