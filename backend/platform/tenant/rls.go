package tenant

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// EnableRLS enables PostgreSQL Row Level Security for all tables that have a tenant_id column.
func EnableRLS(db *gorm.DB, tables ...any) error {
	for _, model := range tables {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			continue
		}

		if stmt.Schema == nil || stmt.Schema.LookUpField("TenantID") == nil {
			continue
		}

		tableName := stmt.Schema.Table

		// Enable RLS and Force RLS
		enableSQL := fmt.Sprintf(`
			ALTER TABLE %s ENABLE ROW LEVEL SECURITY;
			ALTER TABLE %s FORCE ROW LEVEL SECURITY;
			DROP POLICY IF EXISTS tenant_isolation_policy ON %s;
			CREATE POLICY tenant_isolation_policy ON %s AS PERMISSIVE FOR ALL USING (
				current_setting('app.bypass_rls', true) = 'on'
				OR tenant_id = current_setting('app.tenant_id', true)
			);
		`, tableName, tableName, tableName, tableName)

		if err := db.Exec(enableSQL).Error; err != nil {
			return fmt.Errorf("enable rls on %s: %w", tableName, err)
		}
	}
	return nil
}

// SetupAppRole creates a non-superuser database role without BYPASSRLS privilege.
func SetupAppRole(db *gorm.DB, roleName, password string) error {
	sql := fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s WITH LOGIN PASSWORD '%s' NOSUPERUSER NOBYPASSRLS;
			ELSE
				ALTER ROLE %s WITH PASSWORD '%s' NOSUPERUSER NOBYPASSRLS;
			END IF;
		END
		$$;
		GRANT USAGE ON SCHEMA public TO %s;
		GRANT ALL ON ALL TABLES IN SCHEMA public TO %s;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO %s;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO %s;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO %s;
	`, roleName, roleName, password, roleName, password, roleName, roleName, roleName, roleName, roleName)

	return db.Exec(sql).Error
}

// SetDBTenantConfig sets the session-level or transaction-level PostgreSQL configuration variables for RLS.
func SetDBTenantConfig(db *gorm.DB, ctx context.Context) error {
	if filterSkipped(ctx) {
		return db.Exec("SELECT set_config('app.bypass_rls', 'on', false)").Error
	}
	tid := FromContext(ctx)
	return db.Exec("SELECT set_config('app.bypass_rls', 'off', false), set_config('app.tenant_id', ?, false)", tid).Error
}
