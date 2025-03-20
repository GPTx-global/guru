package itest

import (
	"context"
	"fmt"
	"time"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func addMsgUnjailCases() {

	var branch = "v1.0.6"
	var module = "slashing"
	var jailedValIndex = 1

	var node = helpers.NodeInfo{
		Branch:     branch,
		Vals:       4,
		Fulls:      2,
		RunOnIndex: jailedValIndex,
		GenesisParams: []string{
			"app_state.slashing.params.signed_blocks_window=10",
			"app_state.slashing.params.downtime_jail_duration=60s",
		},
	}

	helpers.AddTestCase(&helpers.TestCase{
		// Reset: false,
		Node:   node,
		Module: module,
		Name:   "should pass - unjail",
		Cmd: func(args map[string]interface{}) []string {
			sender := args["sender-address"].(string)
			return cmd.CreateUnjailCmd(sender)
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

			conIdJailedVal, err := m.GetContainerId(branch, jailedValIndex, false)
			if err != nil {
				return nil, err
			}

			// stop the container
			m.KillNode(conIdJailedVal)

			// wait for 10 blocks
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)

			// restart the container
			m.RestartNode(conIdJailedVal)

			// wait two more blocks (for sync)
			m.WaitForNextBlock(ctx, conIdVal0)
			m.WaitForNextBlock(ctx, conIdVal0)

			// wait at least 1 minute before unjailing
			time.Sleep(time.Minute)

			senderAddr, err := helpers.GetAccountByKey(m, &node, "dev0")
			if err != nil {
				return nil, err
			}
			args["sender-address"] = senderAddr

			senderBalance, err := helpers.QueryBalanceOf(m, &node, senderAddr)
			if err != nil {
				return nil, err
			}
			args["sender-balance"] = senderBalance

			validators, err := helpers.QueryValidators(m, &node)
			if err != nil {
				return nil, err
			}
			args["validators"] = validators

			ok := false
			for _, val := range validators {
				if val.Jailed {
					ok = true
				}
			}
			if !ok {
				return nil, fmt.Errorf("jailed validator is not found")
			}

			return args, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			senderAddr := args["sender-address"].(string)
			senderBalance := args["sender-balance"].(sdk.Coins)

			newBalance, err := helpers.QueryBalanceOf(m, &node, senderAddr)
			if err != nil {
				return err
			}

			expectedBalance := senderBalance.Sub(sdk.NewCoin("aguru", sdk.NewInt(18900000000000000)))

			newValidators, err := helpers.QueryValidators(m, &node)
			if err != nil {
				return err
			}

			// check bank module state changes
			if !expectedBalance.IsEqual(newBalance) {
				return fmt.Errorf("bank module state changes failed. expected: %s, got: %s", expectedBalance.String(), newBalance.String())
			}

			// check staking module state changes
			ok := false
			for _, val := range newValidators {
				if val.Jailed {
					ok = true
				}

			}
			if ok {
				return fmt.Errorf("validator is not unjailed")
			}

			return nil
		},
	})

}
