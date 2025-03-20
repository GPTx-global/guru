package itest

import (
	"context"
	"fmt"
	"time"

	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
)

func addDisableInflationCases() {

	var branch = "v1.0.6"
	var module = "inflation"

	var node = helpers.NodeInfo{
		Branch:     branch,
		Vals:       1,
		Fulls:      0,
		RunOnIndex: 0,
		GenesisParams: []string{
			"app_state.inflation.epoch_identifier=minute",
			"app_state.inflation.params.enable_inflation=false",
		},
	}

	helpers.AddTestCase(&helpers.TestCase{
		// Reset: false,
		Node:   node,
		Module: module,
		Name:   "should pass - disable inflation",
		Cmd: func(args map[string]interface{}) []string {
			return []string{}
		},
		ExpPass:    true,
		ExpErr:     "",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			args := make(map[string]interface{})

			conIdVal0, err := m.GetContainerId(branch, 0, false)
			if err != nil {
				return nil, err
			}

			initialTotalSupply, err := helpers.QueryTotalSupply(m, &node)
			if err != nil {
				return nil, err
			}

			// initialCircSupply, err := helpers.QueryCirculatingSupply(m, &node)
			// if err != nil {
			// 	return nil, err
			// }

			// wait at least 2 minutes for inflation taking effect
			time.Sleep(time.Minute * 2)

			// wait one block
			m.WaitForNextBlock(ctx, conIdVal0)

			newTotalSupply, err := helpers.QueryTotalSupply(m, &node)
			if err != nil {
				return nil, err
			}

			// newCircSupply, err := helpers.QueryTotalSupply(m, &node)
			// if err != nil {
			// 	return nil, err
			// }

			if !initialTotalSupply.IsEqual(newTotalSupply) {
				return nil, fmt.Errorf("inflation is not disabled")
			}

			return args, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			return nil
		},
	})

}
