package tgmanager

import (
	"errors"
)

func NewInOutData(chatID, messageID int64, message string, appearType CallBackAppearType, nodes ...NextNode) InOutData {
	out := &inOutData{
		ChatID:     chatID,
		MessageID:  messageID,
		Message:    message,
		AppearType: appearType,
	}
	for i := range nodes {
		out.AddNode(nodes[i])
	}
	return out
}

type InOutData interface {
	AddNode(node NextNode)
	SetMsg(msg string)
	GetMsg() string
	GetChatID() int64
	GetPayload() []byte
	SetPayload(in []byte)
	getMsgID() int64
	generateTelegramContainer() (TelegramContainer, error)
	setMsgID(msgID int64)
	getAppearType() CallBackAppearType
	setDefaultMessage(in string)
	setAppearType(in CallBackAppearType)
}
type inOutData struct {
	ChatID          int64
	MessageID       int64
	Message         string
	ExternalPayload []byte
	AppearType      CallBackAppearType
	ProcessorNodes  []nextNode
	MenuNodes       []nextNode
}

func (i *inOutData) setAppearType(in CallBackAppearType) {
	i.AppearType = in
}

func (i *inOutData) AddNode(node NextNode) {
	if node == nil {
		return
	}
	addNode := node.(*nextNode)

	if inlinePart := node.getInline(); inlinePart != nil {
		i.ProcessorNodes = append(i.ProcessorNodes, *addNode)
		return
	}

	if linkPart := node.getLink(); linkPart != nil {
		i.ProcessorNodes = append(i.ProcessorNodes, *addNode)
		return
	}

	defNode := node.getDefault()
	if defNode == nil {
		return
	}

	if defNode.getProcessorType() == CallbackProcessorTypeProcess {
		defNode.setIdx(len(i.ProcessorNodes))
		i.ProcessorNodes = append(i.ProcessorNodes, *addNode)
		return
	}
	i.MenuNodes = append(i.MenuNodes, *addNode)
}

func (i *inOutData) setDefaultMessage(in string) {
	if i.Message == "" {
		i.Message = in
	}
}

func (i *inOutData) SetMsg(msg string) {
	i.Message = msg
}

func (i *inOutData) GetMsg() string {
	return i.Message
}

func (i *inOutData) GetChatID() int64 {
	return i.ChatID
}

func (i *inOutData) GetPayload() []byte {
	return i.ExternalPayload
}
func (i *inOutData) SetPayload(in []byte) {
	i.ExternalPayload = in
}

func (i *inOutData) setMsgID(msgID int64) {
	i.MessageID = msgID
}

func (i *inOutData) getMsgID() int64 {
	return i.MessageID
}

func (i *inOutData) getAppearType() CallBackAppearType {
	return i.AppearType
}

func (i *inOutData) generateTelegramContainer() (TelegramContainer, error) {
	var tgContainer TelegramContainer
	tgContainer.Buttons = make([]Button, 0, len(i.MenuNodes)+len(i.ProcessorNodes))
	tgContainer.ChatID = i.ChatID
	tgContainer.Message = i.Message
	tgContainer.OldMessageID = i.MessageID
	tgContainer.AppearType = i.AppearType

	for j := range i.ProcessorNodes {
		linNode := i.ProcessorNodes[j].LinkNode
		defNode := i.ProcessorNodes[j].DefaultNode
		inlNode := i.ProcessorNodes[j].InlineNode

		if defNode != nil {
			tgContainer.Buttons = append(tgContainer.Buttons, Button{
				ButtonLabel:   i.ProcessorNodes[j].getButtonLabel(),
				Callback:      defNode.callback(),
				ProcessorType: defNode.getProcessorType(),
			})
		} else if linNode != nil {
			tgContainer.Buttons = append(tgContainer.Buttons, Button{
				ButtonLabel: i.ProcessorNodes[j].getButtonLabel(),
				Link:        linNode.getLink(),
			})
		} else if inlNode != nil {
			tgContainer.Buttons = append(tgContainer.Buttons, Button{
				ButtonLabel:                  i.ProcessorNodes[j].getButtonLabel(),
				SwitchInlineQueryCurrentChat: inlNode.getSwitchInlineQueryCurrentChat(),
			})
		}
	}

	for j := range i.MenuNodes {
		defNode := i.MenuNodes[j].DefaultNode
		if defNode == nil {
			continue
		}
		tgContainer.Buttons = append(tgContainer.Buttons, Button{
			ButtonLabel:   i.MenuNodes[j].getButtonLabel(),
			Callback:      defNode.callback(),
			ProcessorType: defNode.getProcessorType(),
		})
	}

	return tgContainer, nil
}

func (i *inOutData) getProcessorNodeByIndex(idx int64) (nextNode, error) {
	if int(idx) > len(i.ProcessorNodes)-1 {
		return nextNode{}, errors.New("invalid idx")
	}
	return i.ProcessorNodes[idx], nil
}

func (i *inOutData) getMenuNodeByType(processorType CallbackProcessorType) (nextNode, bool) {
	for j := range i.MenuNodes {
		defNode := i.MenuNodes[j].getDefault()
		if defNode == nil {
			return nextNode{}, false
		}
		if defNode.getProcessorType() == processorType {
			return i.MenuNodes[j], true
		}
	}
	return nextNode{}, false
}
