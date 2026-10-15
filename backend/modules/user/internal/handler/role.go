package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"saasbase/modules/user/api"
	"saasbase/platform/authz"
	"saasbase/platform/httpx"
	"saasbase/platform/module"
	"saasbase/platform/tenant"
)

type RoleHandler struct {
	svc     api.RoleServiceServer
	permMgr api.PermissionManager
}

func NewRoleHandler(svc api.RoleServiceServer, permMgr api.PermissionManager) *RoleHandler {
	return &RoleHandler{svc: svc, permMgr: permMgr}
}

func (h *RoleHandler) Register(r gin.IRouter) {
	g := r.Group("/v1/roles")
	g.GET("", authz.Require(api.ResourceRole, "read"), h.list)
	g.POST("", authz.Require(api.ResourceRole, "write"), h.create)
	g.DELETE("/:id", authz.Require(api.ResourceRole, "write"), h.remove)
	g.GET("/:id/permissions", authz.Require(api.ResourceRole, "read"), h.getPermissions)
	g.PUT("/:id/permissions", authz.Require(api.ResourceRole, "write"), h.setPermissions)

	// Catalogue of every permission declared by the loaded modules. The UI needs
	// it to render the grant matrix.
	r.GET("/v1/permissions", authz.Require(api.ResourceRole, "read"), h.catalogue)
}

type createRoleBody struct {
	Name string `json:"name" binding:"required"`
}

type permissionDTO struct {
	Namespace string `json:"namespace"`
	Action    string `json:"action"`
	HostOnly  bool   `json:"host_only,omitempty"`
}

type setPermissionsBody struct {
	Permissions []permissionDTO `json:"permissions"`
}

func (h *RoleHandler) list(c *gin.Context) {
	out, err := h.svc.ListRoles(c.Request.Context(), &api.ListRolesRequest{})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	items := make([]gin.H, 0, len(out.GetItems()))
	for _, r := range out.GetItems() {
		items = append(items, gin.H{"id": r.GetId(), "name": r.GetName(), "preserved": r.GetPreserved()})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RoleHandler) create(c *gin.Context) {
	var body createRoleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.CreateRole(c.Request.Context(), &api.CreateRoleRequest{Name: body.Name})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": out.GetId(), "name": out.GetName()})
}

func (h *RoleHandler) remove(c *gin.Context) {
	if _, err := h.svc.DeleteRole(c.Request.Context(), &api.DeleteRoleRequest{Id: c.Param("id")}); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RoleHandler) getPermissions(c *gin.Context) {
	ctx := c.Request.Context()
	perms, err := h.permMgr.GetRolePermissions(ctx, c.Param("id"), tenant.FromContext(ctx))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	items := make([]permissionDTO, 0, len(perms))
	for _, p := range perms {
		items = append(items, permissionDTO{Namespace: p.Namespace, Action: p.Action})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RoleHandler) setPermissions(c *gin.Context) {
	var body setPermissionsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	perms := make([]authz.Permission, 0, len(body.Permissions))
	for _, p := range body.Permissions {
		perms = append(perms, authz.Permission{Namespace: p.Namespace, Action: p.Action})
	}
	if err := h.permMgr.AssignPermissions(ctx, c.Param("id"), tenant.FromContext(ctx), perms); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": body.Permissions})
}

func (h *RoleHandler) catalogue(c *gin.Context) {
	all := module.Permissions()
	items := make([]permissionDTO, 0, len(all))
	for _, p := range all {
		items = append(items, permissionDTO{Namespace: p.Namespace, Action: p.Action, HostOnly: p.HostOnly})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
