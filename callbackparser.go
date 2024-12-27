package tgmanager

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func newCallback(processor string, processorType CallbackProcessorType) callbackParser {
	return callbackParser{
		Processor:     processor,
		ProcessorType: processorType,
	}
}

const callbackDivider = ">"
const callbackTemp = "%s" + callbackDivider + "%d" + callbackDivider + "%d"

type callbackParser struct {
	Processor     string
	ProcessorType CallbackProcessorType
	Idx           int64
}

func (c *callbackParser) parseCallback(in string) error {
	if in == CallbackProcessorTypeIgnore.String() {
		c.Processor = ""
		c.ProcessorType = CallbackProcessorTypeIgnore
		c.Idx = 0
		return nil
	}
	items := strings.Split(in, callbackDivider)
	if len(items) != 3 {
		return errors.New("invalid callback")
	}

	c.Processor = items[0]

	processorTypeNumber, err := strconv.Atoi(items[1])
	if err != nil {
		return errors.New("invalid callback")
	}

	c.ProcessorType = CallbackProcessorType(processorTypeNumber)
	if !c.ProcessorType.IsValid() {
		return errors.New("invalid callback")
	}

	idxNumber, err := strconv.Atoi(items[2])
	if err != nil || idxNumber < 0 {
		return errors.New("invalid callback")
	}
	c.Idx = int64(idxNumber)
	return nil
}

func (c *callbackParser) setIdx(idx int) {
	c.Idx = int64(idx)
}

func (c *callbackParser) String() string {
	return fmt.Sprintf(callbackTemp, c.Processor, c.ProcessorType, c.Idx)
}
