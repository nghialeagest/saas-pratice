package handler

import (
	"time"

	"saasbase/modules/note/api"
)

// NoteResponse is the HTTP presentation DTO for Note.
// It avoids exposing internal protobuf fields and ensures proper RFC3339 JSON timestamp serialization.
type NoteResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ListNotesResponse represents the paginated response for notes.
type ListNotesResponse struct {
	Items []*NoteResponse `json:"items"`
	Total int64           `json:"total"`
}

// CountNotesResponse represents the count query response.
type CountNotesResponse struct {
	Count int64 `json:"count"`
}

func toNoteResponse(pb *api.Note) *NoteResponse {
	if pb == nil {
		return nil
	}
	var createdAt time.Time
	if pb.CreatedAt != nil {
		createdAt = pb.CreatedAt.AsTime()
	}
	return &NoteResponse{
		ID:        pb.Id,
		Title:     pb.Title,
		Status:    formatStatus(pb.Status),
		CreatedAt: createdAt,
	}
}

func formatStatus(s api.NoteStatus) string {
	switch s {
	case api.NoteStatus_NOTE_STATUS_DRAFT:
		return "DRAFT"
	case api.NoteStatus_NOTE_STATUS_PUBLISHED:
		return "PUBLISHED"
	default:
		return "UNSPECIFIED"
	}
}
