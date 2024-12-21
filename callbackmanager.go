package tgmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/looplab/fsm"
)

const callbackProcessorStorageKey = "callback-processor-msg-id: %d"

type CallbackNodeProcessorFunc func(ctx context.Context, data DefaultDataType) ([]CallbackNode, []byte, error)
type CallbackNotFoundDataProcessorFunc func(ctx context.Context, msgID int64, chatID int64, callback string) error
type MessageNotFoundProcessorFunc func(ctx context.Context, msgID int64, chatID int64, message string) error

type storage interface {
	SaveState(ctx context.Context, key string, data []byte) (err error)
	GetState(ctx context.Context, key string) (data []byte, err error)
	DeleteState(ctx context.Context, key string) (err error)
}

type telegramSender interface {
	SendMsg(ctx context.Context, container TelegramContainer) (msgID int64, err error)
	DeleteMessage(messageID int64, chatID int64)
	GetBotName() (string, error)
}

type CallbackManager interface {
	AddRootCallbackNode(ctx context.Context, opts CallbackOpts) error
	Visualize() string
	NewCallbackNode(opts CallbackOpts) (CallbackNode, error)
	SendNode(ctx context.Context, oldData DefaultDataType) error
	ProcessCallback(ctx context.Context, msgID, chatID int64, callback string) error
	ProcessMsg(ctx context.Context, msgID, chatID int64, callback string) error
}
type callbackManager struct {
	defaultMsg                    string
	defaultAppearType             CallBackAppearType
	defaultCloseProcessor         CallbackNodeProcessorFunc
	defaultBackProcessor          CallbackNodeProcessorFunc
	defaultSkipProcessor          CallbackNodeProcessorFunc
	defaultProcessor              CallbackNodeProcessorFunc
	storage                       storage
	sender                        telegramSender
	callbackDataNotFoundProcessor CallbackNotFoundDataProcessorFunc
	messageNotFoundProcessor      MessageNotFoundProcessorFunc
	rootCallbackNodes             []*callbackNode
	transitionsMap                map[transitionMapItem]interface{}
	callbackNodesMap              map[callbackNodeKey]*callbackNode
	inlineMessagesProcessorsMap   map[string]SwitchInlineProcessorFunc
	botName                       string
}

type transitionMapItem struct {
	src        string
	transition CallbackProcessorType
	dest       string
}

func (c *callbackManager) findAnyNodeByName(name string) *callbackNode {
	arr := []CallbackProcessorType{CallbackProcessorTypeProcess, CallbackProcessorTypeClose,
		CallbackProcessorTypeBack, CallbackProcessorTypeSkip, CallbackProcessorTypeIgnore}
	for i := range arr {
		val, ok := c.callbackNodesMap[callbackNodeKey{
			name:          name,
			processorType: arr[i],
		}]
		if ok {
			return val
		}
	}
	return nil
}

type callbackNodeKey struct {
	name          string
	processorType CallbackProcessorType
}

func NewCallbackManager(
	defaultMsg string,
	defaultAppearType CallBackAppearType,
	defaultProcessor CallbackNodeProcessorFunc,
	storage storage,
	sender telegramSender,
	messageNotFoundProcessor MessageNotFoundProcessorFunc,
	callbackDataNotFoundProcessor CallbackNotFoundDataProcessorFunc,
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
		defaultMsg:                    defaultMsg,
		defaultAppearType:             defaultAppearType,
		defaultProcessor:              defaultProcessor,
		storage:                       storage,
		sender:                        sender,
		callbackNodesMap:              make(map[callbackNodeKey]*callbackNode),
		transitionsMap:                make(map[transitionMapItem]interface{}),
		messageNotFoundProcessor:      messageNotFoundProcessor,
		callbackDataNotFoundProcessor: callbackDataNotFoundProcessor,
		inlineMessagesProcessorsMap:   make(map[string]SwitchInlineProcessorFunc),
		botName:                       botName,
	}, nil
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

func (c *callbackManager) Visualize() string {
	var events = make([]fsm.EventDesc, 0, len(c.callbackNodesMap))

	transCount := 0

	for key, _ := range c.transitionsMap {
		trans := key.transition.String()
		if trans == CallbackProcessorTypeProcess.String() {
			trans = fmt.Sprintf("%s:%d", trans, transCount)
			transCount++
		}
		events = append(events, fsm.EventDesc{
			Name: trans,
			Src:  []string{key.src},
			Dst:  key.dest,
		})
	}
	machine := fsm.NewFSM("root", events, nil)
	return fsm.Visualize(machine)
}

func (c *callbackManager) AddRootCallbackNode(ctx context.Context, opts CallbackOpts) error {
	if !opts.ProcessorType.IsValid() {
		opts.ProcessorType = CallbackProcessorTypeProcess
	}

	node, err := c.newCallbackNode(opts)
	if err != nil {
		return err
	}
	c.rootCallbackNodes = append(c.rootCallbackNodes, node)

	if err = c.processCallbackNode(ctx, "root", CallbackProcessorTypeIgnore, c.rootCallbackNodes[len(c.rootCallbackNodes)-1]); err != nil {
		c.rootCallbackNodes = c.rootCallbackNodes[:len(c.rootCallbackNodes)-1]
		return err
	}
	return nil
}

func (c *callbackManager) NewCallbackNode(opts CallbackOpts) (CallbackNode, error) {
	return c.newCallbackNode(opts)
}

func (c *callbackManager) SendNode(ctx context.Context, oldData DefaultDataType) error {
	oldDataUnwrapped := oldData.(*defaultData)

	tgContainer, newData, err := c.generateTelegramContainer(ctx, 0, oldDataUnwrapped.NodeName, oldDataUnwrapped)
	if err != nil {
		return fmt.Errorf("generate telegram container: %v", err)
	}

	msgID, err := c.sender.SendMsg(ctx, tgContainer)
	if err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}
	newData.MessageID = msgID

	newDataByte, err := json.Marshal(newData)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	if err = c.storage.SaveState(ctx, fmt.Sprintf(callbackProcessorStorageKey, msgID), newDataByte); err != nil {
		return fmt.Errorf("saving to storage: %w", err)
	}
	return nil
}

func (c *callbackManager) ProcessCallback(ctx context.Context, msgID, chatID int64, callback string) error {
	if callback == CallbackProcessorTypeIgnore.String() {
		return nil
	}

	dataByte, err := c.storage.GetState(ctx, fmt.Sprintf(callbackProcessorStorageKey, msgID))
	if err != nil {
		return nil
	}
	if dataByte == nil {
		if c.callbackDataNotFoundProcessor != nil {
			return c.callbackDataNotFoundProcessor(ctx, msgID, chatID, callback)
		}
		return ErrCallbackDataNotFound
	}

	var data defaultData

	if err = json.Unmarshal(dataByte, &data); err != nil {
		return fmt.Errorf("unmarshal storage data: %w", err)
	}

	return c.processCallbackCommandProcess(ctx, msgID, callback, &data)
}

func (c *callbackManager) ProcessMsg(ctx context.Context, msgID, chatID int64, message string) error {
	message = strings.ReplaceAll(message, fmt.Sprintf("@%s", c.botName), "")

	msg, key, userPayload, err := parseSwitchInlineInput(message)
	if err != nil {
		return ErrMessageProcessorNotFound
	}

	processor, ok := c.inlineMessagesProcessorsMap[msg]
	if !ok || processor == nil {
		return ErrMessageProcessorNotFound
	}

	nodeName, payload, err := processor(ctx, key, userPayload)
	if err != nil {
		return fmt.Errorf("processing msg: %w", err)
	}

	if nodeName == "" {
		return nil
	}

	oldData := NewDefaultData(nodeName, chatID, msgID, payload)
	if err = c.SendNode(ctx, oldData); err != nil {
		return fmt.Errorf("sending node: %w", err)
	}
	return nil
}

func (c *callbackManager) addNodeToMap(name string, processorType CallbackProcessorType, node *callbackNode) {
	key := callbackNodeKey{
		name:          name,
		processorType: processorType,
	}
	if _, ok := c.callbackNodesMap[key]; !ok {
		c.callbackNodesMap[key] = node
	}
}

func (c *callbackManager) processCallbackCommandProcess(ctx context.Context, msgID int64, callBack string, oldData *defaultData) error {
	tgContainer, newData, err := c.generateTelegramContainer(ctx, msgID, callBack, oldData)
	if err != nil {
		return fmt.Errorf("generate tg container: %w", err)
	}

	if len(tgContainer.Buttons) == 0 {
		return nil
	}

	msgID, err = c.sender.SendMsg(ctx, tgContainer)
	if err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}
	newData.MessageID = msgID

	newDataByte, err := json.Marshal(newData)
	if err != nil {
		return fmt.Errorf("marhsal data: %w", err)
	}
	if err = c.storage.SaveState(ctx, fmt.Sprintf(callbackProcessorStorageKey, msgID), newDataByte); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

func (c *callbackManager) newCallbackNode(opts CallbackOpts) (*callbackNode, error) {
	if opts.Name == "" {
		return nil, errors.New("callback node name should be specified")
	}
	if opts.SwitchInlineQueryCurrent == nil {
		opts.SwitchInlineQueryCurrent = NewSwitchInlineQueryCurrentChat("", "", nil)
	}

	if !opts.ProcessorType.IsValid() {
		opts.ProcessorType = CallbackProcessorTypeProcess
	}

	if val, ok := c.callbackNodesMap[callbackNodeKey{
		name:          opts.Name,
		processorType: opts.ProcessorType,
	}]; ok {
		return val, nil
	}

	if opts.Message == "" {
		if c.defaultMsg == "" {
			return nil, errors.New("should specify callback node message, or callback node default message")
		}
		opts.Message = c.defaultMsg
	}

	if opts.Processor == nil {
		opts.Processor = c.defaultProcessor
	}
	if opts.AppearType == nil {
		opts.AppearType = &c.defaultAppearType
	}

	return &callbackNode{
		name:                     opts.Name,
		message:                  opts.Message,
		buttonLabel:              opts.ButtonLabel,
		processor:                opts.Processor,
		appearType:               *opts.AppearType,
		processorType:            opts.ProcessorType,
		finalProcessor:           opts.IsFinalProcessor,
		switchInlineQueryCurrent: opts.SwitchInlineQueryCurrent,
	}, nil
}

func (c *callbackManager) processCallbackNode(ctx context.Context, fromNode string, transitionName CallbackProcessorType, node *callbackNode) error {
	if node == nil {
		return nil
	}

	c.transitionsMap[transitionMapItem{
		src:        fromNode,
		transition: transitionName,
		dest:       node.name,
	}] = nil

	if _, ok := c.callbackNodesMap[callbackNodeKey{
		name:          node.name,
		processorType: node.processorType,
	}]; ok {
		return nil
	}

	c.addNodeToMap(node.name, node.processorType, node)

	if node.switchInlineQueryCurrent.Enabled() && node.switchInlineQueryCurrent.getProcessor() != nil {
		c.inlineMessagesProcessorsMap[node.switchInlineQueryCurrent.getMsg()] = node.switchInlineQueryCurrent.getProcessor()
		return nil
	}

	if node.processor == nil {
		return nil
	}
	if node.finalProcessor {
		return nil
	}

	nextNodes, _, err := node.processor(ctx, &defaultData{
		NodeName: node.name,
	})
	if err != nil {
		return err
	}

	for i := range nextNodes {
		nextNode := nextNodes[i].(*callbackNode)
		if err = c.processCallbackNode(ctx, node.name, nextNode.processorType, nextNode); err != nil {
			return err
		}
	}
	return nil
}

func (c *callbackManager) safeRunProcessor(ctx context.Context, processor CallbackNodeProcessorFunc, oldData DefaultDataType) (
	nodes []CallbackNode, newPayload []byte, err error) {
	nodes, newPayload, err = processor(ctx, oldData)
	if err != nil {
		return nil, nil, err
	}

	for i := range nodes {
		c.addNodeToMap(nodes[i].getName(), nodes[i].getProcessorType(), nodes[i].(*callbackNode))
	}
	return nodes, newPayload, nil
}

func (c *callbackManager) generateTelegramContainer(ctx context.Context, msgID int64, nodeName string, oldData *defaultData) (
	tgContainer TelegramContainer, newData *defaultData, err error) {

	node := c.findAnyNodeByName(nodeName)
	if node == nil {
		return TelegramContainer{}, nil, errors.New("node with this name not found")
	}

	if node.processor == nil {
		return TelegramContainer{}, nil, nil
	}

	nodes, newPayload, err := c.safeRunProcessor(ctx, node.processor, DefaultDataType(oldData))
	if err != nil {
		return TelegramContainer{}, nil, err
	}

	if len(nodes) == 0 {
		go func() {
			_ = c.storage.DeleteState(ctx, fmt.Sprintf(callbackProcessorStorageKey, msgID))
		}()
	}

	tgContainer.Buttons = make([]Button, 0, len(nodes))
	tgContainer.ChatID = oldData.ChatID
	tgContainer.OldMessageID = msgID
	tgContainer.AppearType = node.appearType
	tgContainer.Message = node.message

	for i := range nodes {
		tgContainer.Buttons = append(tgContainer.Buttons, Button{
			ButtonLabel:                  nodes[i].getButtonLabel(),
			Callback:                     nodes[i].getName(),
			ProcessorType:                nodes[i].getProcessorType(),
			SwitchInlineQueryCurrentChat: nodes[i].getSwitchInlineQueryCurrentChat(),
		})
	}

	oldData.NodeName = nodeName
	oldData.Payload = newPayload
	return tgContainer, oldData, nil
}
