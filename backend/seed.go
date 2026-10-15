package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	userapi "saasbase/modules/user/api"
	"saasbase/platform/authz"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

// seedDefaultData seeds a rich multi-tenant dataset for local development and auth testing.
func seedDefaultData(ctx context.Context, db *gorm.DB, logger *slog.Logger) {
	internalSvc, err := module.Resolve[userapi.UserInternalServiceServer]()
	if err != nil {
		logger.Warn("seed: failed to resolve user internal service", "error", err)
		return
	}

	userSvc, err := module.Resolve[userapi.UserServiceServer]()
	if err != nil {
		logger.Warn("seed: failed to resolve user service", "error", err)
		return
	}

	roleSvc, err := module.Resolve[userapi.RoleServiceServer]()
	if err != nil {
		logger.Warn("seed: failed to resolve role service", "error", err)
		return
	}

	permMgr, err := module.Resolve[userapi.PermissionManager]()
	if err != nil {
		logger.Warn("seed: failed to resolve permission manager", "error", err)
		return
	}

	// 1. Ensure Tenants with readable IDs: "tenant-default", "tenant-a", "tenant-b"
	ensureTenant := func(id, name, slug string) string {
		var existingID string
		if err := db.Table("tenants").Select("id").Where("slug = ? OR id = ?", slug, id).Scan(&existingID).Error; err == nil && existingID != "" {
			return existingID
		}
		row := map[string]any{
			"id":         id,
			"name":       name,
			"slug":       slug,
			"status":     "ACTIVE",
			"created_at": time.Now(),
			"updated_at": time.Now(),
		}
		_ = db.Table("tenants").Create(&row).Error
		logger.Info("seed: created tenant", "name", name, "slug", slug, "id", id)
		return id
	}

	tenantDefaultID := ensureTenant("tenant-default", "Default Workspace", "default")
	tenantAlphaID := ensureTenant("tenant-a", "Công ty Alpha", "alpha")
	tenantBetaID := ensureTenant("tenant-b", "Công ty Beta", "beta")

	// ensureRole returns the id of the role with the given name inside tenantID,
	// creating it when absent. Guarding per resource instead of per block lets a
	// partially seeded database repair itself on the next start.
	ensureRole := func(tenantID, name string) string {
		normalized := strings.ToUpper(name)
		var existingID string
		_ = db.Table("roles").Select("id").
			Where("tenant_id = ? AND normalized_name = ?", tenantID, normalized).
			Scan(&existingID).Error
		if existingID != "" {
			return existingID
		}
		id := uuid.NewString()
		if err := db.Table("roles").Create(map[string]any{
			"id":              id,
			"name":            name,
			"normalized_name": normalized,
			"tenant_id":       tenantID,
			"created_at":      time.Now(),
			"updated_at":      time.Now(),
		}).Error; err != nil {
			logger.Warn("seed: create role failed", "name", name, "tenant", tenantID, "error", err)
			return ""
		}
		logger.Info("seed: created role", "name", name, "tenant", tenantID)
		return id
	}

	ensureUserRole := func(userID, roleID, tenantID string) {
		if userID == "" || roleID == "" {
			return
		}
		var n int64
		db.Table("user_roles").
			Where("user_id = ? AND role_id = ? AND tenant_id = ?", userID, roleID, tenantID).
			Count(&n)
		if n > 0 {
			return
		}
		_ = db.Table("user_roles").Create(map[string]any{
			"user_id":    userID,
			"role_id":    roleID,
			"tenant_id":  tenantID,
			"created_at": time.Now(),
		}).Error
	}

	ensureUser := func(tenantID, email, username, password string) string {
		var existingID string
		_ = db.Table("users").Select("id").
			Where("normalized_username = ?", strings.ToUpper(username)).
			Scan(&existingID).Error
		if existingID != "" {
			return existingID
		}
		u, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantID,
			Email:    email,
			Username: username,
			Password: password,
		})
		if err != nil || u == nil {
			logger.Warn("seed: provision user failed", "username", username, "error", err)
			return ""
		}
		logger.Info("seed: created user", "username", username, "tenant", tenantID)
		return u.Id
	}

	grantAll := []authz.Permission{{Namespace: "*", Action: "*"}}

	// 2. Account 1: Global Super Admin ("admin" / "password123")
	adminID := ensureUser(tenantDefaultID, "admin@example.com", "admin", "password123")
	if adminID != "" {
		hostRoleID := ensureRole("", "Host Admin")
		ensureUserRole(adminID, hostRoleID, "")
		if hostRoleID != "" {
			_ = permMgr.AssignPermissions(ctx, hostRoleID, "", grantAll)
		}

		_, _ = internalSvc.JoinTenant(ctx, &userapi.JoinTenantRequest{UserId: adminID, TenantId: tenantAlphaID})
		_, _ = internalSvc.JoinTenant(ctx, &userapi.JoinTenantRequest{UserId: adminID, TenantId: tenantBetaID})

		for _, tid := range []string{tenantDefaultID, tenantAlphaID, tenantBetaID} {
			roleID := ensureRole(tid, "Admin")
			ensureUserRole(adminID, roleID, tid)
			if roleID != "" {
				_ = permMgr.AssignPermissions(tenant.NewContext(ctx, tid), roleID, tid, grantAll)
			}
		}
	}

	// 3. Account 2: Tenant A Admin ("alice" / "password123")
	var aliceCount int64
	if db.Table("users").Where("normalized_username = ?", "ALICE").Count(&aliceCount); aliceCount == 0 {
		_, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantAlphaID,
			Email:    "alice@alpha.com",
			Username: "alice",
			Password: "password123",
		})
		if err == nil {
			logger.Info("seed: created alice (Admin in Alpha)", "username", "alice", "tenant", tenantAlphaID)
		}
	}

	// 4. Account 3: Multi-tenant User ("bob" / "password123")
	// Role in Alpha: Editor (note.note:read, note.note:write, task.task:read, task.task:write)
	// Role in Beta: Viewer (note.note:read, task.task:read only - NO write)
	var bobCount int64
	if db.Table("users").Where("normalized_username = ?", "BOB").Count(&bobCount); bobCount == 0 {
		bobUser, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantAlphaID,
			Email:    "bob@example.com",
			Username: "bob",
			Password: "password123",
		})
		if err == nil && bobUser != nil {
			// A. Setup Editor role in tenant Alpha
			ctxAlpha := tenant.NewContext(ctx, tenantAlphaID)
			editorRole, err := roleSvc.CreateRole(ctxAlpha, &userapi.CreateRoleRequest{Name: "Editor"})
			if err == nil && editorRole != nil {
				_, _ = userSvc.AssignRoles(ctxAlpha, &userapi.AssignRolesRequest{
					UserId:  bobUser.Id,
					RoleIds: []string{editorRole.Id},
				})
				_ = permMgr.AssignPermissions(ctxAlpha, editorRole.Id, tenantAlphaID, []authz.Permission{
					{Namespace: "note.note", Action: "read"},
					{Namespace: "note.note", Action: "write"},
					{Namespace: "task.task", Action: "read"},
					{Namespace: "task.task", Action: "write"},
				})
			}

			// B. Join Bob to tenant Beta and setup Viewer role (Read-only)
			_, _ = internalSvc.JoinTenant(ctx, &userapi.JoinTenantRequest{UserId: bobUser.Id, TenantId: tenantBetaID})
			ctxBeta := tenant.NewContext(ctx, tenantBetaID)
			viewerRole, err := roleSvc.CreateRole(ctxBeta, &userapi.CreateRoleRequest{Name: "Viewer"})
			if err == nil && viewerRole != nil {
				_, _ = userSvc.AssignRoles(ctxBeta, &userapi.AssignRolesRequest{
					UserId:  bobUser.Id,
					RoleIds: []string{viewerRole.Id},
				})
				_ = permMgr.AssignPermissions(ctxBeta, viewerRole.Id, tenantBetaID, []authz.Permission{
					{Namespace: "note.note", Action: "read"},
					{Namespace: "task.task", Action: "read"},
				})
			}
			logger.Info("seed: created bob (Editor in Alpha, Viewer in Beta)", "username", "bob")
		}
	}

	// 5. Account 4: Single-tenant User ("charlie" / "password123" - Beta ONLY)
	var charlieCount int64
	if db.Table("users").Where("normalized_username = ?", "CHARLIE").Count(&charlieCount); charlieCount == 0 {
		_, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantBetaID,
			Email:    "charlie@beta.com",
			Username: "charlie",
			Password: "password123",
		})
		if err == nil {
			logger.Info("seed: created charlie (Beta only)", "username", "charlie", "tenant", tenantBetaID)
		}
	}

	// 6. Account 5: Locked Account ("david" / "password123")
	var davidCount int64
	if db.Table("users").Where("normalized_username = ?", "DAVID").Count(&davidCount); davidCount == 0 {
		davidUser, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantDefaultID,
			Email:    "david@example.com",
			Username: "david",
			Password: "password123",
		})
		if err == nil && davidUser != nil {
			lockout := time.Now().Add(24 * time.Hour)
			db.Table("users").Where("id = ?", davidUser.Id).Updates(map[string]any{
				"access_failed_count": 5,
				"lockout_until":       lockout,
			})
			logger.Info("seed: created david (Locked out)", "username", "david")
		}
	}

	// 7. Account 6: Inactive Account ("eva" / "password123")
	var evaCount int64
	if db.Table("users").Where("normalized_username = ?", "EVA").Count(&evaCount); evaCount == 0 {
		evaUser, err := internalSvc.ProvisionTenantAdmin(ctx, &userapi.ProvisionTenantAdminRequest{
			TenantId: tenantDefaultID,
			Email:    "eva@example.com",
			Username: "eva",
			Password: "password123",
		})
		if err == nil && evaUser != nil {
			db.Table("users").Where("id = ?", evaUser.Id).Update("status", "INACTIVE")
			logger.Info("seed: created eva (Inactive)", "username", "eva")
		}
	}
}
