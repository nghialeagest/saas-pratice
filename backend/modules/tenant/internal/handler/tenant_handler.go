package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"saasbase/modules/tenant/api"
	"saasbase/platform/authz"
	"saasbase/platform/httpx"
)

type TenantHandler struct {
	svc api.TenantServiceServer
}

func NewTenantHandler(svc api.TenantServiceServer) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) Register(r gin.IRouter) {
	g := r.Group("/v1/tenants")
	g.POST("", authz.Require(api.ResourceTenant, "write"), h.create)
	g.GET("", authz.Require(api.ResourceTenant, "read"), h.list)
	g.GET("/:id", authz.Require(api.ResourceTenant, "read"), h.get)
	g.PUT("/:id", authz.Require(api.ResourceTenant, "write"), h.update)
}

func (h *TenantHandler) create(c *gin.Context) {
	var req api.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *TenantHandler) get(c *gin.Context) {
	id := c.Param("id")
	out, err := h.svc.GetTenant(c.Request.Context(), &api.GetTenantRequest{ID: id})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *TenantHandler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	out, err := h.svc.ListTenants(c.Request.Context(), &api.ListTenantsRequest{
		Page:     page,
		PageSize: size,
		Status:   status,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *TenantHandler) update(c *gin.Context) {
	var req api.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	req.ID = c.Param("id")
	out, err := h.svc.UpdateTenant(c.Request.Context(), &req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}
