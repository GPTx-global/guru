package itest

import (
	"context"
	"fmt"
	"strings"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"

	"github.com/GPTx-global/guru/tests/itest/docker"
)

func addChangeModeratorCases() {
	var branch = "v1.0.6"
	var module = "distribution"

	helpers.AddTestCase(&helpers.TestCase{
		Node: helpers.NodeInfo{
			Branch: branch,
			Vals:   1,
			Fulls:  0,
		},
		Module: module,
		Name:   "should pass - change the moderator",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["old-moderator"].(string)
			newModerator := args["new-moderator"].(string)
			return cmd.CreateChangeModeratorCmd(moderator, newModerator)
		},
		ExpPass:    true,
		ExpErr:     "",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			currentModAddr, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}

			dev0Addr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "dev0")
			if err != nil {
				return nil, err
			}
			modAddr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "modkey")
			if err != nil {
				return nil, err
			}
			initialValues["old-moderator"] = modAddr
			initialValues["new-moderator"] = dev0Addr

			if currentModAddr == dev0Addr {
				initialValues["old-moderator"] = dev0Addr
				initialValues["new-moderator"] = modAddr
			}

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			oldModerator := args["old-moderator"].(string)
			newModerator := args["new-moderator"].(string)

			currentMdoerator, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return err
			}

			if currentMdoerator != newModerator {
				return fmt.Errorf("moderator address should have been changed. expected: %s, got: %s", newModerator, currentMdoerator)
			}

			conId, err := m.GetContainerId(branch, 0, false)
			if err != nil {
				return err
			}
			exec, err := m.CreateExec(cmd.CreateChangeModeratorCmd(newModerator, oldModerator), conId)
			if err != nil {
				return err
			}

			outBuf, errBuf, err := m.RunExec(ctx, exec)
			if err != nil || errBuf.String() != "" {
				return err
			}

			if !strings.Contains(outBuf.String(), "code: 0") {
				return fmt.Errorf("tx returned non code 0:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String())
			}
			return nil
		},
	})

	helpers.AddTestCase(&helpers.TestCase{
		Node: helpers.NodeInfo{
			Branch: branch,
			Vals:   1,
			Fulls:  0,
		},
		Module: module,
		Name:   "should fail - wrong sender",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["old-moderator"].(string)
			newModerator := args["new-moderator"].(string)
			return cmd.CreateChangeModeratorCmd(newModerator, moderator)
		},
		ExpPass:    false,
		ExpErr:     "only moderator is allowed",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			currentModAddr, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}

			dev0Addr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "dev0")
			if err != nil {
				return nil, err
			}
			modAddr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "modkey")
			if err != nil {
				return nil, err
			}
			initialValues["old-moderator"] = modAddr
			initialValues["new-moderator"] = dev0Addr

			if currentModAddr == dev0Addr {
				initialValues["old-moderator"] = dev0Addr
				initialValues["new-moderator"] = modAddr
			}

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			return nil
		},
	})
}
