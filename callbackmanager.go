package tgmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/looplab/fsm"
)

const (
	transitionOpen              = "open"
	transitionProcess           = "process"
	transitionBack              = "back"
	transitionClose             = "close"
	transitionSkip              = "skip"
	stateEnd                    = "%s: END"
	stateClose                  = "%s: CLOSE"
	stateSkip                   = "%s: SKIP"
	stateBack                   = "%s: BACK"
	callbackProcessorStorageKey = "callback-processor-msg-id: %d"

	callbackCommandProcess = "process"
	callbackCommandSkip    = "skip"
	callbackCommandBack    = "back"
	callbackCommandClose   = "close"
)

type CallbackNodeProcessorFunc func(ctx context.Context, payload []byte) ([]CallbackNode, []byte, error)
type CallbackNodeActionsProcessorFunc func(ctx context.Context) (CallbackNode, error)

type CallbackManager interface {
	AddRootCallbackNode(ctx context.Context, opts CallbackOpts) error
	Visualize() string
	NewCallbackNode(opts CallbackOpts) (CallbackNode, error)
}

type storage interface {
	SaveState(key string, data []byte) (err error)
	GetState(key string) (data []byte, err error)
}

type telegramSender interface {
	SendMsg(ctx context.Context, container TelegramContainer, msgToUpdate int64) (msgID int64, err error)
	DeleteMsg(msgID int64) (err error)
}

type storageData struct {
	NodeName string
	Payload  []byte
}

type callbackManager struct {
	defaultMsg            string
	defaultAppearType     CallBackAppearType
	defaultCloseProcessor CallbackNodeActionsProcessorFunc
	defaultBackProcessor  CallbackNodeActionsProcessorFunc
	defaultSkipProcessor  CallbackNodeActionsProcessorFunc
	defaultProcessor      CallbackNodeProcessorFunc
	rootCallbackNodes     []*callbackNode
	transitionArr         []transition
	tempTransitionsArr    []transition
	callbackNodesMap      map[string]*callbackNode
	storage               storage
	sender                telegramSender
}

type CallbackManagerSettings struct {
	DefaultMsg            string
	DefaultAppearType     CallBackAppearType
	DefaultCloseProcessor CallbackNodeActionsProcessorFunc
	DefaultBackProcessor  CallbackNodeActionsProcessorFunc
	DefaultSkipProcessor  CallbackNodeActionsProcessorFunc
	DefaultProcessor      CallbackNodeProcessorFunc
	storage               storage
}

type TelegramContainer struct {
}

func NewCallbackManager(opts CallbackManagerSettings) (CallbackManager, error) {
	if opts.DefaultMsg == "" {
		return nil, errors.New("default msg is required field")
	}
	return &callbackManager{
		defaultMsg:            opts.DefaultMsg,
		defaultAppearType:     opts.DefaultAppearType,
		defaultCloseProcessor: opts.DefaultCloseProcessor,
		defaultBackProcessor:  opts.DefaultBackProcessor,
		defaultSkipProcessor:  opts.DefaultSkipProcessor,
		defaultProcessor:      opts.DefaultProcessor,
		storage:               opts.storage,
	}, nil
}

type transition struct {
	fromNode       string
	transitionName string
	toNode         string
}

func (c *callbackManager) Visualize() string {
	var events = make([]fsm.EventDesc, 0, len(c.transitionArr))

	for i := range c.transitionArr {
		events = append(events, fsm.EventDesc{
			Name: c.transitionArr[i].transitionName,
			Src:  []string{c.transitionArr[i].fromNode},
			Dst:  c.transitionArr[i].toNode,
		})
	}

	machine := fsm.NewFSM("root", events, nil)
	return fsm.Visualize(machine)
}

func (c *callbackManager) AddRootCallbackNode(ctx context.Context, opts CallbackOpts) error {
	node, err := c.newCallbackNode(opts)
	if err != nil {
		return err
	}
	c.rootCallbackNodes = append(c.rootCallbackNodes, node)

	defer func() {
		c.transitionArr = append(c.transitionArr, c.tempTransitionsArr...)
		c.tempTransitionsArr = nil
	}()

	if err = c.processCallbackNode(ctx, "root", transitionOpen, c.rootCallbackNodes[len(c.rootCallbackNodes)-1]); err != nil {
		c.rootCallbackNodes = c.rootCallbackNodes[:len(c.rootCallbackNodes)-1]
		return err
	}
	return nil
}

func (c *callbackManager) NewCallbackNode(opts CallbackOpts) (CallbackNode, error) {
	node, err := c.newCallbackNode(opts)
	return node, err
}

func (c *callbackManager) SendNode(ctx context.Context, name string, oldPayload []byte) error {
	node, ok := c.callbackNodesMap[name]
	if !ok {
		return errors.New("node with this name not found")
	}

	tgContainer, newPayload, err := c.generateTelegramContainer(ctx, node, oldPayload)
	if err != nil {
		return fmt.Errorf("generate telegram container: %v", err)
	}

	storagePayload, err := wrapPayload(node.name, newPayload)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	msgID, err := c.sender.SendMsg(ctx, tgContainer, 0)
	if err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}

	if err = c.storage.SaveState(fmt.Sprintf(callbackProcessorStorageKey, msgID), storagePayload); err != nil {
		return fmt.Errorf("saving to storage: %w", err)
	}
	return nil
}

func (c *callbackManager) ProcessCallback(ctx context.Context, msgID int64, callback string) error {
	data, err := c.storage.GetState(fmt.Sprintf(callbackProcessorStorageKey, msgID))
	if err != nil {
		return fmt.Errorf("getting state from db: %w", err)
	}

	payload, _, err := unwrapPayload(data)
	if err != nil {
		return fmt.Errorf("unwrap payload: %w", err)
	}

	//node, ok := c.callbackNodesMap[nodeName]
	//if !ok {
	//	return fmt.Errorf("node not found")
	//}

	if callback == callbackCommandProcess {
		return c.processCallbackCommandProcess(ctx, callback, payload)
	}
	return nil
}

func (c *callbackManager) processCallbackCommandProcess(ctx context.Context, callBack string, oldPayload []byte) error {
	node, ok := c.callbackNodesMap[callBack]
	if !ok {
		return fmt.Errorf("node not found")
	}

	tgContainer, newPayload, err := c.generateTelegramContainer(ctx, node, oldPayload)
	if err != nil {
		return fmt.Errorf("generate tg container: %w", err)
	}

	msgID, err := c.sender.SendMsg(ctx, tgContainer, 0)
	if err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}

	if err = c.storage.SaveState(fmt.Sprintf(callbackProcessorStorageKey, msgID), newPayload); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

func (c *callbackManager) newCallbackNode(opts CallbackOpts) (*callbackNode, error) {
	if opts.Name == "" {
		return nil, errors.New("callback node name should be specified")
	}

	if opts.Message == "" {
		if c.defaultMsg == "" {
			return nil, errors.New("should specify callback node message, or callback node default message")
		}
		opts.Message = c.defaultMsg
	}

	if opts.CloseProcessor == nil {
		opts.CloseProcessor = c.defaultCloseProcessor
	}
	if opts.BackProcessor == nil {
		opts.BackProcessor = c.defaultBackProcessor
	}
	if opts.SkipProcessor == nil {
		opts.SkipProcessor = c.defaultSkipProcessor
	}
	if opts.Processor == nil {
		opts.Processor = c.defaultProcessor
	}
	if opts.AppearType == nil {
		opts.AppearType = &c.defaultAppearType
	}
	return &callbackNode{
		name:           opts.Name,
		message:        opts.Message,
		buttonLabel:    opts.ButtonLabel,
		closeProcessor: opts.CloseProcessor,
		backProcessor:  opts.BackProcessor,
		skipProcessor:  opts.SkipProcessor,
		processor:      opts.Processor,
		appearType:     *opts.AppearType,
	}, nil
}

func (c *callbackManager) processCallbackNode(ctx context.Context, fromNode, transitionName string, node *callbackNode) error {
	if node == nil {
		return nil
	}

	c.callbackNodesMap[node.name] = node

	c.tempTransitionsArr = append(c.tempTransitionsArr, transition{
		fromNode:       fromNode,
		transitionName: transitionName,
		toNode:         node.name,
	})

	if node.processor == nil {
		return nil
	}

	nextNodes, _, err := node.processor(ctx, nil)
	if err != nil {
		return err
	}

	for i := range nextNodes {
		if err = c.processCallbackNode(ctx, node.name, transitionProcess, nextNodes[i].(*callbackNode)); err != nil {
			return err
		}
	}

	if node.closeProcessor != nil {
		closeNode, err := node.closeProcessor(ctx)
		if err != nil {
			return err
		}
		if closeNode == nil {
			c.tempTransitionsArr = append(c.tempTransitionsArr, transition{
				fromNode:       node.name,
				transitionName: transitionClose,
				toNode:         fmt.Sprintf(stateClose, node.name),
			})
		} else {
			if err = c.processCallbackNode(ctx, node.name, transitionClose, closeNode.(*callbackNode)); err != nil {
				return err
			}
		}
	}

	if node.backProcessor != nil {
		backNode, err := node.backProcessor(ctx)
		if err != nil {
			return err
		}
		if backNode == nil {
			c.tempTransitionsArr = append(c.tempTransitionsArr, transition{
				fromNode:       node.name,
				transitionName: transitionBack,
				toNode:         fmt.Sprintf(stateBack, node.name),
			})
		} else {
			if err = c.processCallbackNode(ctx, node.name, transitionBack, backNode.(*callbackNode)); err != nil {
				return err
			}
		}
	}

	if node.skipProcessor != nil {
		skipNode, err := node.skipProcessor(ctx)
		if err != nil {
			return err
		}
		if skipNode == nil {
			c.tempTransitionsArr = append(c.tempTransitionsArr, transition{
				fromNode:       node.name,
				transitionName: transitionSkip,
				toNode:         fmt.Sprintf(stateSkip, node.name),
			})
		} else {
			if err = c.processCallbackNode(ctx, node.name, transitionSkip, skipNode.(*callbackNode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *callbackManager) safeRunProcessor(ctx context.Context, processor CallbackNodeProcessorFunc, payload []byte) (nodes []CallbackNode, outPayload []byte, err error) {
	nodes, outPayload, err = processor(ctx, payload)
	if err != nil {
		return nil, nil, err
	}

	for i := range nodes {
		if _, ok := c.callbackNodesMap[nodes[i].Name()]; !ok {
			c.callbackNodesMap[nodes[i].Name()] = nodes[i].(*callbackNode)
		}
	}
	return nodes, outPayload, nil
}

func (c *callbackManager) generateTelegramContainer(ctx context.Context, node *callbackNode, oldPayload []byte) (tgContainer TelegramContainer, newPayload []byte, err error) {
	_, newPayload, err = c.safeRunProcessor(ctx, node.processor, oldPayload)
	if err != nil {
		return TelegramContainer{}, nil, err
	}

	return TelegramContainer{}, newPayload, nil
}
