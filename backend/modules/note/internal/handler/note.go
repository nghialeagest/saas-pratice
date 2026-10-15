package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"saasbase/modules/note/api"
	"saasbase/platform/authz"
	"saasbase/platform/httpx"
)

type Handler struct {
	svc api.NoteServiceServer
}

func New(svc api.NoteServiceServer) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r gin.IRouter) {
	g := r.Group("/v1/notes")
	g.POST("", authz.Require(api.Resource, "write"), h.create)
	g.GET("", authz.Require(api.Resource, "read"), h.list)
	g.GET("/count", authz.Require(api.Resource, "read"), h.count)
	g.GET("/:id", authz.Require(api.Resource, "read"), h.get)
	g.POST("/:id/publish", authz.Require(api.Resource, "write"), h.publish)
}

type createBody struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body"`
}

func (h *Handler) create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, err)
		return
	}
	out, err := h.svc.CreateNote(c.Request.Context(), &api.CreateNoteRequest{
		Title: body.Title,
		Body:  body.Body,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toNoteResponse(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.svc.GetNote(c.Request.Context(), &api.GetNoteRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toNoteResponse(out))
}

func (h *Handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListNotes(c.Request.Context(), &api.ListNotesRequest{
		Status:   parseStatus(c.Query("status")),
		Page:     int32(page),
		PageSize: int32(size),
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}

	items := make([]*NoteResponse, len(out.GetItems()))
	for i, item := range out.GetItems() {
		items[i] = toNoteResponse(item)
	}

	c.JSON(http.StatusOK, &ListNotesResponse{
		Items: items,
		Total: out.GetTotal(),
	})
}

func (h *Handler) count(c *gin.Context) {
	out, err := h.svc.CountNotes(c.Request.Context(), &api.CountNotesRequest{
		Status: parseStatus(c.Query("status")),
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, &CountNotesResponse{Count: out.GetCount()})
}

func (h *Handler) publish(c *gin.Context) {
	out, err := h.svc.PublishNote(c.Request.Context(), &api.PublishNoteRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toNoteResponse(out))
}

func parseStatus(s string) api.NoteStatus {
	switch s {
	case "DRAFT":
		return api.NoteStatus_NOTE_STATUS_DRAFT
	case "PUBLISHED":
		return api.NoteStatus_NOTE_STATUS_PUBLISHED
	default:
		return api.NoteStatus_NOTE_STATUS_UNSPECIFIED
	}
}
