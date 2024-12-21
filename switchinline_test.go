package tgmanager

import (
	"errors"
	"fmt"
	"testing"
)

func Test_getBracketsInOut(t *testing.T) {
	type testContainer struct {
		Text string
		In   string
		Out  string
	}

	for _, tCase := range []struct {
		name  string
		input testContainer
	}{
		{
			name: "1",
			input: testContainer{
				Text: "some",
				In:   "",
				Out:  "some",
			},
		},
		{
			name: "2",
			input: testContainer{
				Text: "some(",
				In:   "",
				Out:  "some(",
			},
		},
		{
			name: "3",
			input: testContainer{
				Text: "some(puki)",
				In:   "puki",
				Out:  "some",
			},
		},
	} {
		in, out := getBracketsInOut(tCase.input.Text)
		if tCase.input.In != in {
			t.Error("in not match")
		}
		if tCase.input.Out != out {
			t.Error("out not match")
		}
	}
}

func Test_parseSwitchInlineInput(t *testing.T) {
	type testContainer struct {
		Msg     string
		Key     string
		Payload string
		Err     error
	}

	for _, tCase := range []struct {
		name     string
		rawInput string
		expected testContainer
	}{
		{
			name:     "1",
			rawInput: fmt.Sprintf("some%ssome", inlineDivider),
			expected: testContainer{
				Msg:     "some",
				Key:     "",
				Payload: "some",
				Err:     nil,
			},
		},
		{
			name:     "2",
			rawInput: fmt.Sprintf("some       (kaka)%ssome", inlineDivider),
			expected: testContainer{
				Msg:     "some",
				Key:     "kaka",
				Payload: "some",
				Err:     nil,
			},
		},
		{
			name:     "3",
			rawInput: fmt.Sprintf("some       (kaka)%ssome", "invalid"),
			expected: testContainer{
				Msg:     "some",
				Key:     "kaka",
				Payload: "some",
				Err:     errors.New("some"),
			},
		},
		{
			name:     "4",
			rawInput: fmt.Sprintf("some%s      some      ", inlineDivider),
			expected: testContainer{
				Msg:     "some",
				Key:     "",
				Payload: "some",
				Err:     nil,
			},
		},
	} {
		msg, key, payload, err := parseSwitchInlineInput(tCase.rawInput)
		if (err == nil) != (tCase.expected.Err == nil) {
			t.Error("errors non match")
		}
		if err == nil {
			if msg != tCase.expected.Msg ||
				key != tCase.expected.Key ||
				payload != tCase.expected.Payload {
				if msg != tCase.expected.Msg {
					t.Error("msg non match", "actual:", msg, "expected:", tCase.expected.Msg)
				}
				if key != tCase.expected.Key {
					t.Error("key non match", "actual:", key, "expected:", tCase.expected.Key)
				}
				if payload != tCase.expected.Payload {
					t.Error("payload non match", "actual:", payload, "expected:", tCase.expected.Payload)
				}
			}
		}
	}
}
