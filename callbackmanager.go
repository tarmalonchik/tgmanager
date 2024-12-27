package tgmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type CallbackNodeProcessorFunc func(ctx context.Context, data InOutData) (InOutData, error)
type CallbackNotFoundDataProcessorFunc func(ctx context.Context, msgID int64, chatID int64, callback string) error
type MessageNotFoundProcessorFunc func(ctx context.Context, msgID int64, chatID int64, message string) error

type storage interface {
	SaveState(ctx context.Context, key int64, data []byte) (err error)
	GetState(ctx context.Context, key int64) (data []byte, err error)
	DeleteState(ctx context.Context, key int64) (err error)
}

type telegramSender interface {
	SendMsg(ctx context.Context, container TelegramContainer) (msgID int64, err error)
	DeleteMessage(messageID int64, chatID int64)
	GetBotName() (string, error)
}

type logger interface {
	LogError(err error)
}

type CallbackManager interface {
	SendNode(ctx context.Context, data InOutData, processor string) error
	ProcessCallback(ctx context.Context, oldMsgID, chatID int64, callback string) error
	ProcessMsg(ctx context.Context, msgID, chatID int64, callback string) error
	AddProcessors(items ...Processor) error
	AddInlineProcessors(items ...InlineProcessor) error
	GetProcessor(name string) CallbackNodeProcessorFunc
}
type callbackManager struct {
	defaultMsg                    string
	defaultAppearType             CallBackAppearType
	defaultCloseProcessor         CallbackNodeProcessorFunc
	defaultBackProcessor          CallbackNodeProcessorFunc
	defaultSkipProcessor          CallbackNodeProcessorFunc
	defaultProcessor              CallbackNodeProcessorFunc
	allProcessors                 map[string]CallbackNodeProcessorFunc
	storage                       storage
	sender                        telegramSender
	callbackDataNotFoundProcessor CallbackNotFoundDataProcessorFunc
	messageNotFoundProcessor      MessageNotFoundProcessorFunc
	inlineProcessorMap            map[string]SwitchInlineProcessorFunc
	botName                       string
	logger                        logger
}

func (c *callbackManager) findAnyProcessorByName(name string) (CallbackNodeProcessorFunc, error) {
	if val, ok := c.allProcessors[name]; !ok {
		return nil, errors.New("processor not found")
	} else {
		return val, nil
	}
}

func NewCallbackManager(
	defaultMsg string,
	defaultAppearType CallBackAppearType,
	defaultProcessor CallbackNodeProcessorFunc,
	storage storage,
	sender telegramSender,
	messageNotFoundProcessor MessageNotFoundProcessorFunc,
	callbackDataNotFoundProcessor CallbackNotFoundDataProcessorFunc,
	logger logger,
) (CallbackManager, error) {
	if defaultMsg == "" {
		return nil, errors.New("default msg is required field")
	}

	if storage == nil || sender == nil {
		return nil, errors.New("storage and telegramSender are required")
	}

	botName, err := sender.GetBotName()
	if err != nil {
		return nil, errors.New("getting bot name")
	}

	return &callbackManager{
		defaultMsg:        defaultMsg,
		defaultAppearType: defaultAppearType,
		defaultProcessor:  defaultProcessor,
		storage:           storage,
		sender:            sender,
		//callbackNodesMap:              make(map[callbackNodeKey]*CallbackNode),
		//transitionsMap:                make(map[transitionMapItem]interface{}),
		messageNotFoundProcessor:      messageNotFoundProcessor,
		callbackDataNotFoundProcessor: callbackDataNotFoundProcessor,
		inlineProcessorMap:            make(map[string]SwitchInlineProcessorFunc),
		botName:                       botName,
		logger:                        logger,
	}, nil
}

type InlineProcessor struct {
	Name      string
	Processor SwitchInlineProcessorFunc
}

func (c *callbackManager) AddInlineProcessors(items ...InlineProcessor) error {
	for i := range items {
		if err := c.addInlineProcessor(items[i].Name, items[i].Processor); err != nil {
			return err
		}
	}
	return nil
}

func (c *callbackManager) GetProcessor(name string) CallbackNodeProcessorFunc {
	if val, ok := c.allProcessors[name]; ok {
		return val
	}
	return nil
}

func (c *callbackManager) addInlineProcessor(name string, processor SwitchInlineProcessorFunc) error {
	if c.inlineProcessorMap == nil {
		c.inlineProcessorMap = make(map[string]SwitchInlineProcessorFunc)
	}
	if _, ok := c.inlineProcessorMap[name]; ok {
		return errors.New(fmt.Sprintf("duplicate processor: %s", name))
	}
	c.inlineProcessorMap[name] = processor
	return nil
}

type Processor struct {
	Name      string
	Processor CallbackNodeProcessorFunc
}

func (c *callbackManager) AddProcessors(items ...Processor) error {
	for i := range items {
		if err := c.addProcessor(items[i].Name, items[i].Processor); err != nil {
			return err
		}
	}
	return nil
}

func (c *callbackManager) addProcessor(name string, processor CallbackNodeProcessorFunc) error {
	if c.allProcessors == nil {
		c.allProcessors = make(map[string]CallbackNodeProcessorFunc)
	}
	if _, ok := c.allProcessors[name]; ok {
		return errors.New(fmt.Sprintf("duplicate processor: %s", name))
	}
	c.allProcessors[name] = processor
	return nil
}

func (c *callbackManager) SetDefaultProcessor(defaultProcessor CallbackNodeProcessorFunc) {
	c.defaultProcessor = defaultProcessor
}

func (c *callbackManager) SetMessageNotFoundProcessor(messageNotFoundProcessor MessageNotFoundProcessorFunc) {
	c.messageNotFoundProcessor = messageNotFoundProcessor
}

func (c *callbackManager) SetCallbackDataNotFoundProcessor(callbackDataNotFoundProcessor CallbackNotFoundDataProcessorFunc) {
	c.callbackDataNotFoundProcessor = callbackDataNotFoundProcessor
}

func (c *callbackManager) SendNode(ctx context.Context, data InOutData, processor string) error {
	if data == nil {
		return fmt.Errorf("data is required data")
	}

	if data.getAppearType() != CallBackAppearTypeResend {
		return fmt.Errorf("this method support only %s appear type", CallBackAppearTypeResend.String())
	}

	proc, ok := c.allProcessors[processor]
	if !ok {
		return errors.New("processor not found")
	}

	newData, err := proc(ctx, data)
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}
	if newData == nil {
		newData = data
	}
	newData.setDefaultMessage(c.defaultMsg)

	tgCont, err := newData.generateTelegramContainer()
	if err != nil {
		return fmt.Errorf("generate container: %w", err)
	}

	newMsgID, err := c.sender.SendMsg(ctx, tgCont)
	if err != nil {
		return fmt.Errorf("sending tg msg: %w", err)
	}

	newData.setMsgID(newMsgID)
	if err = c.setDataToStorage(ctx, newData); err != nil {
		return fmt.Errorf("save data to storage: %w", err)
	}
	return nil
}

func (c *callbackManager) ProcessMsg(ctx context.Context, msgID, chatID int64, message string) error {
	message = strings.ReplaceAll(message, fmt.Sprintf("@%s", c.botName), "")

	msg, key, userPayload, err := parseSwitchInlineInput(message)
	if err != nil {
		return ErrMessageProcessorNotFound
	}

	processor, ok := c.inlineProcessorMap[msg]
	if !ok || processor == nil {
		return ErrMessageProcessorNotFound
	}

	outData, processorName, err := processor(ctx, key, userPayload, chatID, msgID)
	if err != nil {
		return fmt.Errorf("processing msg: %w", err)
	}
	if outData == nil {
		return nil
	}

	if err = c.SendNode(ctx, outData, processorName); err != nil {
		return fmt.Errorf("sending node: %w", err)
	}
	return nil
}

func (c *callbackManager) ProcessCallback(ctx context.Context, oldMsgID, chatID int64, callbackValue string) error {
	var callback callbackParser
	if err := callback.parseCallback(callbackValue); err != nil {
		return errors.New("invalid callback")
	}

	if callback.ProcessorType == CallbackProcessorTypeIgnore {
		return nil
	}

	data, err := c.dataProcessor(ctx, oldMsgID, callback)
	if err != nil {
		return err
	}
	if data == nil {
		c.clearFlow(ctx, oldMsgID, chatID)
		return nil
	}

	tgContainer, err := data.generateTelegramContainer()
	if err != nil {
		return err
	}

	newMsgID, err := c.sender.SendMsg(ctx, tgContainer)
	if err != nil {
		return err
	}
	data.setMsgID(newMsgID)

	if data.getAppearType() == CallBackAppearTypeResendDeleteOld {
		c.deleteDataFromStorage(ctx, oldMsgID)
	}

	if err = c.setDataToStorage(ctx, data); err != nil {
		return err
	}
	return nil
}

func (c *callbackManager) deleteDataFromStorage(ctx context.Context, msgID int64) {
	go func() {
		if err := c.storage.DeleteState(ctx, msgID); err != nil {
			if c.logger != nil {
				c.logger.LogError(err)
			}
		}
	}()
}

func (c *callbackManager) getDataFromStorage(ctx context.Context, msgID int64) (InOutData, error) {
	payload, err := c.storage.GetState(ctx, msgID)
	if err != nil {
		return nil, fmt.Errorf("getting data from storage: %w", err)
	}

	var data inOutData

	if err = json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("json unmarshal")
	}
	return &data, nil
}

func (c *callbackManager) setDataToStorage(ctx context.Context, data InOutData) error {
	dataStruct, ok := data.(*inOutData)
	if !ok {
		return errors.New("invalid inbound data")
	}

	dataPayload, err := json.Marshal(dataStruct)
	if err != nil {
		return errors.New("json marshal")
	}

	if err = c.storage.SaveState(ctx, data.getMsgID(), dataPayload); err != nil {
		return fmt.Errorf("saving data to storage: %w", err)
	}
	return nil
}

func (c *callbackManager) dataProcessor(ctx context.Context, msgID int64, callback callbackParser) (InOutData, error) {
	payload, err := c.storage.GetState(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, nil
	}

	var data = &inOutData{}

	if err = json.Unmarshal(payload, data); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	var nxtNode nextNode
	if callback.ProcessorType == CallbackProcessorTypeProcess {
		if nxtNode, err = data.getProcessorNodeByIndex(callback.Idx); err != nil {
			return nil, errors.New("invalid index")
		}
	} else {
		var valid bool
		nxtNode, valid = data.getMenuNodeByType(callback.ProcessorType)
		if !valid {
			return nil, errors.New("invalid non processor type callback")
		}
	}

	defNextNode := nxtNode.getDefault()
	if defNextNode == nil {
		return nil, errors.New("invalid next node")
	}

	if defNextNode.getProcessorName() == "" {
		return nil, nil
	}

	processor, ok := c.allProcessors[defNextNode.getProcessorName()]
	if !ok {
		return nil, errors.New("processor not found")
	}
	if processor == nil {
		return nil, nil
	}

	data.ExternalPayload = defNextNode.getExternalPayload()
	data.MenuNodes = nil
	data.ProcessorNodes = nil

	newData, err := processor(ctx, data)
	if err != nil {
		return nil, err
	}
	if newData != nil {
		newData.setAppearType(c.defaultAppearType)
	}

	return newData, nil
}

func (c *callbackManager) clearFlow(ctx context.Context, msgID, chatID int64) {
	go func() {
		c.sender.DeleteMessage(msgID, chatID)
		c.deleteDataFromStorage(ctx, msgID)
	}()
}
