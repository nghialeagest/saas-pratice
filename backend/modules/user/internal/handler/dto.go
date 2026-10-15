package handler

import (
	"time"

	"saasbase/modules/user/api"
)

type roleDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userDTO struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Name           string    `json:"name,omitempty"`
	Email          string    `json:"email"`
	EmailConfirmed bool      `json:"email_confirmed"`
	Gender         string    `json:"gender,omitempty"`
	Roles          []roleDTO `json:"roles"`
	CreatedAt      time.Time `json:"created_at"`
}

type userListDTO struct {
	Items []userDTO `json:"items"`
	Total int64     `json:"total"`
}

func toUserDTO(u *api.User) userDTO {
	if u == nil {
		return userDTO{}
	}
	roles := make([]roleDTO, 0, len(u.GetRoles()))
	for _, r := range u.GetRoles() {
		roles = append(roles, roleDTO{ID: r.GetId(), Name: r.GetName()})
	}
	out := userDTO{
		ID:             u.GetId(),
		Username:       u.GetUsername(),
		Name:           u.GetName(),
		Email:          u.GetEmail(),
		EmailConfirmed: u.GetEmailConfirmed(),
		Gender:         u.GetGender().String(),
		Roles:          roles,
	}
	if ts := u.GetCreatedAt(); ts != nil {
		out.CreatedAt = ts.AsTime()
	}
	return out
}

func toUserListDTO(res *api.ListUsersResponse) userListDTO {
	if res == nil {
		return userListDTO{Items: []userDTO{}}
	}
	items := make([]userDTO, 0, len(res.GetItems()))
	for _, u := range res.GetItems() {
		items = append(items, toUserDTO(u))
	}
	return userListDTO{Items: items, Total: res.GetTotal()}
}
