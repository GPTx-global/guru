package itest

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func addMsgDelegateCases() {

	var branch = "v1.0.6"
	var module = "staking"

	var node = helpers.NodeInfo{
		Branch:     branch,
		Vals:       2,
		Fulls:      2,
		RunOnIndex: 0,
	}

	helpers.AddTestCase(&helpers.TestCase{
		// Reset: false,
		Node:   node,
		Module: module,
		Name:   "should pass - new delegation",
		Cmd: func(args map[string]interface{}) []string {
			sender := args["sender-address"].(string)
			validator := args["validators"].([]helpers.Validator)
			return cmd.CreateDelegateCmd(sender, validator[0].OperatorAddress, "1000000000000000000000aguru")
		},
		ExpPass:    true,
		ExpErr:     "",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			args := make(map[string]interface{})

			senderAddr, err := helpers.GetAccountByKey(m, &node, "tester1")
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

			return args, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			senderAddr := args["sender-address"].(string)
			validators := args["validators"].([]helpers.Validator)
			senderBalance := args["sender-balance"].(sdk.Coins)

			conId, err := m.GetContainerId(branch, 0, true)
			if err != nil {
				return err
			}

			newBalance, err := helpers.QueryBalanceOf(m, &node, senderAddr)
			if err != nil {
				return err
			}

			delAmount, ok := sdk.NewIntFromString("1000000000000000000000")
			if !ok {
				return fmt.Errorf("cannot convert %s to sdk.Int", "1000000000000000000000")
			}
			expectedBalance := senderBalance.Sub(sdk.NewCoin("aguru", sdk.NewInt(50400000000000000))).Sub(sdk.NewCoin("aguru", delAmount))

			newValidators, err := helpers.QueryValidators(m, &node)
			if err != nil {
				return err
			}
			operatorAddr := validators[0].OperatorAddress
			oldTotalPower := math.NewInt(0)
			newTotalPower := math.NewInt(0)
			oldDelegations := math.NewInt(0)
			newDelegations := math.NewInt(0)
			for _, val := range validators {
				oldTotalPower = oldTotalPower.Add(val.Tokens)
				if val.OperatorAddress == operatorAddr {
					oldDelegations = val.Tokens
				}
			}
			for _, val := range newValidators {
				newTotalPower = newTotalPower.Add(val.Tokens)
				if val.OperatorAddress == operatorAddr {
					newDelegations = val.Tokens
				}
			}

			// check bank module state changes
			if !expectedBalance.IsEqual(newBalance) {
				return fmt.Errorf("bank module state changes failed. expected: %s, got: %s", expectedBalance.String(), newBalance.String())
			}

			// check the staking module state changes
			if !oldDelegations.Add(delAmount).Equal(newDelegations) {
				return fmt.Errorf("staking module state changes failed")
			}

			// check the gov module state changes
			if !oldTotalPower.Add(delAmount).Equal(newTotalPower) {
				return fmt.Errorf("gov module state changes failed")
			}

			// check the distribution module state changes
			err = helpers.ExecSend(m, &node, senderAddr, "guru1un3w727nmqzhvg0jeg24rtzzqpnyjqnl8auuty", "10aguru")
			if err != nil {
				return err
			}
			m.WaitForNextBlock(ctx, conId)
			delegationRewards, err := helpers.QueryRewardsAll(m, &node, senderAddr)
			if err != nil {
				return err
			}

			if delegationRewards.Empty() {
				return fmt.Errorf("distribution module state changes failed")
			}

			return nil
		},
	})
}
