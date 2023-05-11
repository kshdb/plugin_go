// Copyright 2015 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package plugin_go

import (
	"errors"
	"strings"
)

const (
	errorCodeConnFailed = "err-connection-failed"
	errorCodeHttpServe  = "err-http-serve"
)

// 连接到外部插件失败的出错信息
type ErrConnectionFailed error

// 外部插件无法开始侦听调用时的出错信息
type ErrHttpServe error

// 外部插件打印无效消息时的出错信息
type ErrInvalidMessage error

// 插件注册失败时的出错信息
type ErrRegistrationTimeout error

func parseError(line string) error {
	parts := strings.SplitN(line, ": ", 2)
	if parts[0] == "" {
		return nil
	}

	err := errors.New(parts[1])

	switch parts[0] {
	case errorCodeConnFailed:
		return ErrConnectionFailed(err)
	case errorCodeHttpServe:
		return ErrHttpServe(err)
	}

	return err
}
