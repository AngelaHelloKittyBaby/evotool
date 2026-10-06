package service

import "errors"

// ErrMissingDependency 表示服务初始化或运行时缺少必需依赖。
var ErrMissingDependency = errors.New("missing service dependency")
