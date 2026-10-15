package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"saasbase/modules/task/api"
	"saasbase/platform/authz"
	"saasbase/platform/httpx"
)

type Handler struct {
	svc api.TaskServiceServer
}

func New(svc api.TaskServiceServer) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r gin.IRouter) {
	g := r.Group("/v1/tasks")
	g.POST("", authz.Require(api.Resource, "write"), h.create)
	g.GET("/count", authz.Require(api.Resource, "read"), h.count)
	g.GET("/:id", authz.Require(api.Resource, "read"), h.get)
}

type createBody struct {
	Title  string `json:"title" binding:"required"`
	NoteID string `json:"note_id"`
}

func (h *Handler) create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.CreateTask(c.Request.Context(), &api.CreateTaskRequest{
		Title:  body.Title,
		NoteId: body.NoteID,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTaskResponse(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.svc.GetTask(c.Request.Context(), &api.GetTaskRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(out))
}

func (h *Handler) count(c *gin.Context) {
	out, err := h.svc.CountTasks(c.Request.Context(), &api.CountTasksRequest{})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, &CountTasksResponse{Count: out.GetCount()})
}
