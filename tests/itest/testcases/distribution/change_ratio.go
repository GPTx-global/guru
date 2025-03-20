package itest

import (
	"context"
	"fmt"
	"strings"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"

	"github.com/GPTx-global/guru/tests/itest/docker"
)

func addChangeRatioCases() {
	var branch = "v1.0.6"
	var module = "distribution"

	helpers.AddTestCase(&helpers.TestCase{
		Node: helpers.NodeInfo{
			Branch:        branch,
			Vals:          1,
			Fulls:         0,
			RunOnFullnode: false,
		},
		Module: module,
		Name:   "should pass - change fee ratio params",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["current-moderator"].(string)
			return cmd.CreateChangeRatioCmd(moderator, "0.34", "0.33", "0.33")
		},
		ExpPass: true,
		ExpErr:  "",
		CheckCases: []helpers.CheckCase{
			{
				Module:   "distribution",
				Query:    "ratio",
				Args:     []string{},
				Expected: "ratio:\n  base: \"0.330000000000000000\"\n  burn: \"0.330000000000000000\"\n  staking_rewards: \"0.340000000000000000\"\n",
			},
		},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			currentModAddr, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			initialValues["current-moderator"] = currentModAddr

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			moderator := args["current-moderator"].(string)
			conId, err := m.GetContainerId(branch, 0, false)
			if err != nil {
				return err
			}
			exec, err := m.CreateExec(cmd.CreateChangeRatioCmd(moderator, "0.333333333333333334", "0.333333333333333333", "0.333333333333333333"), conId)
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
		Name:   "should fail - wrong ratio params",
		Cmd: func(args map[string]interface{}) []string {
			moderator := args["current-moderator"].(string)
			return cmd.CreateChangeRatioCmd(moderator, "0.33", "0.33", "0.33")
		},
		ExpPass:    false,
		ExpErr:     "the ratio should sum up to be 1.0",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})

			currentModAddr, err := helpers.QueryModeratorAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			initialValues["current-moderator"] = currentModAddr

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
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
			return cmd.CreateChangeRatioCmd("basekey", "0.34", "0.33", "0.33")
		},
		ExpPass:    false,
		ExpErr:     "only moderator is allowed",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			initialValues := make(map[string]interface{})
			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			return nil
		},
	})
}
