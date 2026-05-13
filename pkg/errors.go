package pkg

import "errors"

// ErrorEnums 定义项目级错误
var (
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrInstanceNotFound = errors.New("instance not found")
	ErrPluginNotFound  = errors.New("plugin not found")
	ErrInvalidParam    = errors.New("invalid param")
	ErrInternal        = errors.New("internal error")
)
