package tgmanager

type TelegramContainer struct {
	ChatID       int64
	OldMessageID int64
	Message      string
	AppearType   CallBackAppearType
	Buttons      []Button
}

func (t *TelegramContainer) GetButtonByProcessorType(processorType CallbackProcessorType) (Button, bool) {
	for i := range t.Buttons {
		if t.Buttons[i].ProcessorType == processorType {
			return t.Buttons[i], true
		}
	}
	return Button{}, false
}

type Button struct {
	ButtonLabel                  string
	Callback                     string
	ProcessorType                CallbackProcessorType
	SwitchInlineQueryCurrentChat SwitchInlineQueryCurrentChat
	Link                         Link
}

type link struct {
	link string
}

func (l *link) GetLink() string {
	return l.link
}

type Link interface {
	GetLink() string
}
