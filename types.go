//go:generate go-enum -f=$GOFILE --nocase
package tgmanager

import (
	"errors"
)

var (
	ErrMessageProcessorNotFound = errors.New("message processor not found")
)

// CallBackAppearType ENUM(update,resend,resend_delete_old)
type CallBackAppearType string

// CallbackProcessorType ENUM(process,back,close,skip,ignore)
type CallbackProcessorType int
