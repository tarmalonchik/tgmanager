package tgmanager

type CallbackNode interface {
	Processor() CallbackNodeProcessorFunc
	getName() string
	getButtonLabel() string
	getProcessorType() CallbackProcessorType
	getSwitchInlineQueryCurrentChat() SwitchInlineQueryCurrentChat
}

type CallbackOpts struct {
	Name                     string
	Message                  string
	ButtonLabel              string
	Processor                CallbackNodeProcessorFunc
	AppearType               *CallBackAppearType
	ProcessorType            CallbackProcessorType
	IsFinalProcessor         bool
	SwitchInlineQueryCurrent SwitchInlineQueryCurrentChat
}

type callbackNode struct {
	name                     string
	message                  string
	buttonLabel              string
	processor                CallbackNodeProcessorFunc
	appearType               CallBackAppearType
	processorType            CallbackProcessorType
	finalProcessor           bool
	switchInlineQueryCurrent SwitchInlineQueryCurrentChat
}

func (c *callbackNode) Processor() CallbackNodeProcessorFunc {
	return c.processor
}
func (c *callbackNode) getProcessorType() CallbackProcessorType {
	return c.processorType
}
func (c *callbackNode) getName() string {
	return c.name
}
func (c *callbackNode) getButtonLabel() string {
	return c.buttonLabel
}
func (c *callbackNode) getSwitchInlineQueryCurrentChat() SwitchInlineQueryCurrentChat {
	return c.switchInlineQueryCurrent
}
