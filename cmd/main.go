package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/tarmalonchik/tgmanager"
)

func main() {
	ctx := context.Background()

	manager, err := tgmanager.NewCallbackManager(tgmanager.CallbackManagerSettings{
		DefaultMsg: "msg",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	err = manager.AddRootCallbackNode(ctx, tgmanager.CallbackOpts{
		Name:           "Admin",
		Processor:      getAdminProcessor(manager),
		CloseProcessor: getAdminCloseProcessor(manager),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	visualize(manager.Visualize())
}

func getAdminProcessor(manager tgmanager.CallbackManager) tgmanager.CallbackNodeProcessorFunc {
	return func(ctx context.Context) ([]tgmanager.CallbackNode, error) {
		node, err := manager.NewCallbackNode(tgmanager.CallbackOpts{
			Name: "kaka",
		})
		if err != nil {
			return nil, err
		}
		return []tgmanager.CallbackNode{node}, nil
	}
}

func getAdminCloseProcessor(manager tgmanager.CallbackManager) tgmanager.CallbackNodeActionsProcessorFunc {
	return func(ctx context.Context) (tgmanager.CallbackNode, error) {
		return nil, nil
		node, err := manager.NewCallbackNode(tgmanager.CallbackOpts{
			Name: "kaka",
		})
		if err != nil {
			return nil, err
		}
		return []tgmanager.CallbackNode{node}, nil
	}
}

func visualize(in string) {
	if err := os.WriteFile("assets/temp", []byte(in), 0666); err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		_ = os.Remove("assets/temp")
	}()
	if err := exec.Command("bash", "-c", "dot -Tsvg assets/temp > assets/tg_state_machine.svg").Run(); err != nil {
		fmt.Println(err)
	}
}
