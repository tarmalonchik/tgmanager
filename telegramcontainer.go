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
}
