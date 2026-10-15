package api

import "saasbase/platform/module"

const Resource = "task.task"

func Permissions() []module.Permission {
	return []module.Permission{
		{Namespace: Resource, Action: "read"},
		{Namespace: Resource, Action: "write"},
	}
}
