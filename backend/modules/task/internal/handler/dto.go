package handler

import (
	"time"

	"saasbase/modules/task/api"
)

// TaskResponse is the HTTP presentation DTO for Task.
// It avoids exposing internal protobuf fields and ensures proper RFC3339 JSON timestamp serialization.
type TaskResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	NoteID    string    `json:"note_id,omitempty"`
	NoteTitle string    `json:"note_title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CountTasksResponse represents the count query response.
type CountTasksResponse struct {
	Count int64 `json:"count"`
}

func toTaskResponse(pb *api.Task) *TaskResponse {
	if pb == nil {
		return nil
	}
	var createdAt time.Time
	if pb.CreatedAt != nil {
		createdAt = pb.CreatedAt.AsTime()
	}
	return &TaskResponse{
		ID:        pb.Id,
		Title:     pb.Title,
		NoteID:    pb.NoteId,
		NoteTitle: pb.NoteTitle,
		CreatedAt: createdAt,
	}
}
