package api

import "saasbase/platform/module"

const Resource = "note.note"

func Permissions() []module.Permission {
	return []module.Permission{
		{Namespace: Resource, Action: "read"},
		{Namespace: Resource, Action: "write"},
		{Namespace: Resource, Action: "*", HostOnly: true},
	}
}
