package itest

import (
	"context"
	"fmt"
	"strings"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"

	"github.com/GPTx-global/guru/tests/itest/docker"
)

func addChangeBaseAddressCases() {
	var branch = "v1.0.6"
	var module = "distribution"

	helpers.AddTestCase(&helpers.TestCase{
		Node: helpers.NodeInfo{
			Branch: branch,
			Vals:   1,
			Fulls:  0,
		},
		Module: module,
		Name:   "should pass - change the base address",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["moderator"].(string)
			newBaseAddr := args["new-base-address"].(string)
			return cmd.CreateChangeBaseAddressCmd(moderator, newBaseAddr)
		},
		ExpPass:    true,
		ExpErr:     "",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			moderator, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			initialValues["moderator"] = moderator

			currentBaseAddr, err := helpers.QueryBaseAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}

			dev0Addr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "dev0")
			if err != nil {
				return nil, err
			}
			baseAddr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "basekey")
			if err != nil {
				return nil, err
			}
			initialValues["old-base-address"] = baseAddr
			initialValues["new-base-address"] = dev0Addr

			if currentBaseAddr == dev0Addr {
				initialValues["old-base-address"] = dev0Addr
				initialValues["new-base-address"] = baseAddr
			}

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			moderator := args["moderator"].(string)
			newBaseAddr := args["new-base-address"].(string)
			oldBaseAddr := args["old-base-address"].(string)

			currentBaseAddr, err := helpers.QueryBaseAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return err
			}

			if currentBaseAddr != newBaseAddr {
				return fmt.Errorf("base address should have been changed. expected: %s, got: %s", newBaseAddr, currentBaseAddr)
			}

			conId, err := m.GetContainerId(branch, 0, false)
			if err != nil {
				return err
			}
			exec, err := m.CreateExec(cmd.CreateChangeBaseAddressCmd(moderator, oldBaseAddr), conId)
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
			Branch:        branch,
			Vals:          1,
			Fulls:         0,
			RunOnFullnode: false,
		},
		Module: module,
		Name:   "should fail - wrong sender ()",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["moderator"].(string)
			newBaseAddr := args["new-base-address"].(string)
			return cmd.CreateChangeBaseAddressCmd(newBaseAddr, moderator)
		},
		ExpPass:    false,
		ExpErr:     "only moderator is allowed",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			moderator, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			initialValues["moderator"] = moderator

			currentBaseAddr, err := helpers.QueryBaseAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}

			dev0Addr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "dev0")
			if err != nil {
				return nil, err
			}
			baseAddr, err := helpers.GetAccountByKey(m, &helpers.NodeInfo{Branch: branch}, "basekey")
			if err != nil {
				return nil, err
			}
			initialValues["old-base-address"] = baseAddr
			initialValues["new-base-address"] = dev0Addr

			if currentBaseAddr == dev0Addr {
				initialValues["old-base-address"] = dev0Addr
				initialValues["new-base-address"] = baseAddr
			}

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			return nil
		},
	})

}
