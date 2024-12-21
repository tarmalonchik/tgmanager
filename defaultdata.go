package tgmanager

func NewDefaultData(nodeName string, chatID, msgID int64, payload []byte) DefaultDataType {
	return &defaultData{
		NodeName:  nodeName,
		ChatID:    chatID,
		MessageID: msgID,
		Payload:   payload,
	}
}

type defaultData struct {
	NodeName  string
	ChatID    int64
	MessageID int64
	Payload   []byte
}

func (d *defaultData) GetPayload() []byte {
	return d.Payload
}
func (d *defaultData) GetChatID() int64 {
	return d.ChatID
}
func (d *defaultData) GetNodeName() string {
	return d.NodeName
}
func (d *defaultData) GetMessageID() int64 {
	return d.MessageID
}
func (d *defaultData) SetPayload(payload []byte) {
	d.Payload = payload
}

type DefaultDataType interface {
	GetPayload() []byte
	GetChatID() int64
	GetNodeName() string
	GetMessageID() int64
	SetPayload(payload []byte)
}
