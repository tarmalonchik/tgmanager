package tgmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type SwitchInlineProcessorFunc func(ctx context.Context, key, userPayload string, chatID, msgID int64) (data InOutData, processorName string, err error)

type SwitchInlineQueryCurrentChat interface {
	GetText() string
	getMsg() string
}
type switchInlineQueryCurrentChat struct {
	msg string
	key string
}

const inlineDivider = "\n→"

func (s *switchInlineQueryCurrentChat) GetText() string {
	if s.key != "" {
		return fmt.Sprintf("%s (%s) %s ", s.msg, s.key, inlineDivider)
	}
	return fmt.Sprintf("%s%s ", s.msg, inlineDivider)
}

func (s *switchInlineQueryCurrentChat) getMsg() string {
	return s.msg
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
