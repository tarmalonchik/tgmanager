package tgmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type SwitchInlineProcessorFunc func(ctx context.Context, key, userPayload string) (nodeName string, payload []byte, err error)

type switchInlineQueryCurrentChat struct {
	Enable    bool
	msg       string
	Key       string
	processor SwitchInlineProcessorFunc
}

const inlineDivider = "\n→"

func (s *switchInlineQueryCurrentChat) Enabled() bool {
	return s.Enable
}
func (s *switchInlineQueryCurrentChat) getMsg() string {
	return s.msg
}
func (s *switchInlineQueryCurrentChat) getProcessor() SwitchInlineProcessorFunc {
	return s.processor
}

func (s *switchInlineQueryCurrentChat) GetText() string {
	if !s.Enable {
		return ""
	}
	if s.Key != "" {
		return fmt.Sprintf("%s (%s) %s ", s.msg, s.Key, inlineDivider)
	}
	return fmt.Sprintf("%s%s ", s.msg, inlineDivider)
}

type SwitchInlineQueryCurrentChat interface {
	GetText() string
	Enabled() bool
	getProcessor() SwitchInlineProcessorFunc
	getMsg() string
}

// NewSwitchInlineQueryCurrentChat
// message and processor are required fields
func NewSwitchInlineQueryCurrentChat(message, key string, processor SwitchInlineProcessorFunc) SwitchInlineQueryCurrentChat {
	if message == "" || processor == nil {
		return &switchInlineQueryCurrentChat{}
	}
	return &switchInlineQueryCurrentChat{
		Enable:    true,
		msg:       message,
		Key:       key,
		processor: processor,
	}
}

func parseSwitchInlineInput(in string) (msg, key, payload string, err error) {
	items := strings.Split(in, inlineDivider)
	if len(items) != 2 {
		return "", "", "", errors.New("invalid input")
	}

	key, msg = getBracketsInOut(items[0])
	return strings.TrimSpace(msg), strings.TrimSpace(key), strings.TrimSpace(items[1]), nil
}

func getBracketsInOut(input string) (msg, key string) {
	openIdx := strings.Index(input, "(")
	closeIdx := strings.Index(input, ")")
	if !(openIdx >= 0 && closeIdx >= 0) || openIdx > closeIdx {
		return "", input
	}
	return input[openIdx+1 : closeIdx], input[:openIdx] + input[closeIdx+1:]
}
