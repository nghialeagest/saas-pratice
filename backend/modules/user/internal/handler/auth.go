package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"saasbase/modules/user/api"
	"saasbase/platform/authn"
	"saasbase/platform/httpx"
)

type AuthHandler struct {
	svc api.AuthServiceServer
}

func NewAuthHandler(svc api.AuthServiceServer) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(r gin.IRouter) {
	g := r.Group("/v1/auth")
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
	g.POST("/logout", h.logout)
}

type loginRequest struct {
	UsernameOrEmail string `json:"username_or_email" binding:"required"`
	Password        string `json:"password" binding:"required"`
	TenantID        string `json:"tenant_id"`
}

func (h *AuthHandler) login(c *gin.Context) {
	var body loginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &api.LoginRequest{
		UsernameOrEmail: body.UsernameOrEmail,
		Password:        body.Password,
		TenantId:        body.TenantID,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) refresh(c *gin.Context) {
	var body refreshRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	resp, err := h.svc.Refresh(c.Request.Context(), &api.RefreshRequest{
		RefreshToken: body.RefreshToken,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

type logoutRequest struct {
	SessionID string `json:"session_id"`
}

func (h *AuthHandler) logout(c *gin.Context) {
	var body logoutRequest
	_ = c.ShouldBindJSON(&body)

	// If session_id is not passed in body, fallback to Identity from context
	sid := body.SessionID
	if sid == "" {
		if id, ok := authn.FromContext(c.Request.Context()); ok {
			sid = id.SessionID
		}
	}

	resp, err := h.svc.Logout(c.Request.Context(), &api.LogoutRequest{
		SessionId: sid,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
