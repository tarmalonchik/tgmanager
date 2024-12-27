package tgmanager

func NewDefaultNode(buttonLabel, processorName string, processorType CallbackProcessorType, externalPayload []byte) NextNode {
	return &nextNode{
		ButtonLabel: buttonLabel,
		DefaultNode: &defaultNode{
			ProcessorName:   processorName,
			ExternalPayload: externalPayload,
			CallbackParser:  newCallback(processorName, processorType),
		},
	}
}

func NewInlineNode(buttonLabel, message, key string) NextNode {
	return &nextNode{
		ButtonLabel: buttonLabel,
		InlineNode: &inlineNode{
			Message: message,
			Key:     key,
		},
	}
}

func NewLinkNode(buttonLabel, link string) NextNode {
	return &nextNode{
		ButtonLabel: buttonLabel,
		LinkNode: &linkNode{
			Link: link,
		},
	}
}

type NextNode interface {
	getButtonLabel() string
	getInline() *inlineNode
	getDefault() *defaultNode
	getLink() *linkNode
}

type nextNode struct {
	ButtonLabel string
	DefaultNode *defaultNode
	InlineNode  *inlineNode
	LinkNode    *linkNode
}

type linkNode struct {
	Link string
}

func (n *nextNode) getInline() *inlineNode {
	return n.InlineNode
}
func (n *nextNode) getDefault() *defaultNode {
	return n.DefaultNode
}
func (n *nextNode) getLink() *linkNode { return n.LinkNode }

type defaultNode struct {
	ProcessorName   string
	ExternalPayload []byte
	CallbackParser  callbackParser
}

func (n *defaultNode) setIdx(idx int) {
	n.CallbackParser.setIdx(idx)
}

type inlineNode struct {
	Message string
	Key     string
}

func (i *inlineNode) getSwitchInlineQueryCurrentChat() SwitchInlineQueryCurrentChat {
	if i.Message == "" {
		return &switchInlineQueryCurrentChat{}
	}
	return &switchInlineQueryCurrentChat{
		msg: i.Message,
		key: i.Key,
	}
}

func (i *linkNode) getLink() Link {
	return &link{
		link: i.Link,
	}
}

func (n *defaultNode) callback() string {
	return n.CallbackParser.String()
}

func (n *defaultNode) getProcessorName() string {
	return n.ProcessorName
}

func (n *defaultNode) getExternalPayload() []byte {
	return n.ExternalPayload
}

func (n *defaultNode) getProcessorType() CallbackProcessorType {
	return n.CallbackParser.ProcessorType
}

func (n *nextNode) getButtonLabel() string {
	return n.ButtonLabel
}
