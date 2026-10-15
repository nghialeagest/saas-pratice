package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"saasbase/modules/user/api"
	"saasbase/platform/authz"
	"saasbase/platform/httpx"
)

type UserHandler struct {
	svc api.UserServiceServer
}

func NewUserHandler(svc api.UserServiceServer) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(r gin.IRouter) {
	g := r.Group("/v1/users")
	g.GET("", authz.Require(api.ResourceUser, "read"), h.list)
	g.POST("", authz.Require(api.ResourceUser, "write"), h.create)
	g.GET("/:id", authz.Require(api.ResourceUser, "read"), h.get)
	g.PUT("/:id", authz.Require(api.ResourceUser, "write"), h.update)
	g.DELETE("/:id", authz.Require(api.ResourceUser, "write"), h.remove)
	g.PUT("/:id/roles", authz.Require(api.ResourceUser, "write"), h.assignRoles)
}

type createUserBody struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
}

type updateUserBody struct {
	Name string `json:"name"`
}

type assignRolesBody struct {
	RoleIDs []string `json:"role_ids"`
}

func (h *UserHandler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListUsers(c.Request.Context(), &api.ListUsersRequest{
		Page:     int32(page),
		PageSize: int32(size),
		Search:   c.Query("search"),
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserListDTO(out))
}

func (h *UserHandler) get(c *gin.Context) {
	out, err := h.svc.GetUser(c.Request.Context(), &api.GetUserRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(out))
}

func (h *UserHandler) create(c *gin.Context) {
	var body createUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.CreateUser(c.Request.Context(), &api.CreateUserRequest{
		Username: body.Username,
		Email:    body.Email,
		Password: body.Password,
		Name:     body.Name,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserDTO(out))
}

func (h *UserHandler) update(c *gin.Context) {
	var body updateUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.UpdateUser(c.Request.Context(), &api.UpdateUserRequest{
		Id:   c.Param("id"),
		Name: body.Name,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(out))
}

func (h *UserHandler) remove(c *gin.Context) {
	if _, err := h.svc.DeleteUser(c.Request.Context(), &api.DeleteUserRequest{Id: c.Param("id")}); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) assignRoles(c *gin.Context) {
	var body assignRolesBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.AssignRoles(c.Request.Context(), &api.AssignRolesRequest{
		UserId:  c.Param("id"),
		RoleIds: body.RoleIDs,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(out))
}
