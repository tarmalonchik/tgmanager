package tgmanager

type CallbackNode interface {
	Name() string
}

type callbackNode struct {
	name           string
	message        string
	buttonLabel    string
	closeProcessor CallbackNodeActionsProcessorFunc
	backProcessor  CallbackNodeActionsProcessorFunc
	skipProcessor  CallbackNodeActionsProcessorFunc
	processor      CallbackNodeProcessorFunc
	appearType     CallBackAppearType
}

func (c *callbackNode) Name() string {
	return c.name
}

type CallbackOpts struct {
	Name           string
	Message        string
	ButtonLabel    string
	CloseProcessor CallbackNodeActionsProcessorFunc
	BackProcessor  CallbackNodeActionsProcessorFunc
	SkipProcessor  CallbackNodeActionsProcessorFunc
	Processor      CallbackNodeProcessorFunc
	AppearType     *CallBackAppearType
}
