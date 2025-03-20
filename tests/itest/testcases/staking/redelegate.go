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

func addMsgRedelegateCases() {

	var branch = "v1.0.6"
	var module = "staking"
	var redelegationAmount = "500000000000000000000"

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
		Name:   "should pass - redelegation",
		Cmd: func(args map[string]interface{}) []string {
			sender := args["sender-address"].(string)
			validatorFrom := args["validator-from"].(helpers.Validator)
			validatorTo := args["validator-to"].(helpers.Validator)
			return cmd.CreateRedelegateCmd(sender, validatorFrom.OperatorAddress, validatorTo.OperatorAddress, redelegationAmount+"aguru")
		},
		ExpPass:    true,
		ExpErr:     "",
		CheckCases: []helpers.CheckCase{},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			args := make(map[string]interface{})

			senderAddr, err := helpers.GetAccountByKey(m, &node, "dev0")
			if err != nil {
				return nil, err
			}
			args["sender-address"] = senderAddr

			delegations, err := helpers.QueryDelegations(m, &node, senderAddr)
			if err != nil {
				return nil, err
			}
			if len(delegations) == 0 {
				return nil, fmt.Errorf("address does not have any delegations")
			}
			valFromAddr := delegations[0].Delegation.ValidatorAddress

			senderBalance, err := helpers.QueryBalanceOf(m, &node, senderAddr)
			if err != nil {
				return nil, err
			}
			args["sender-balance"] = senderBalance

			senderRewards, err := helpers.QueryRewards(m, &node, senderAddr, valFromAddr)
			if err != nil {
				return nil, err
			}
			rewardsInInt, _ := senderRewards.TruncateDecimal()
			args["sender-rewards"] = rewardsInInt

			validators, err := helpers.QueryValidators(m, &node)
			if err != nil {
				return nil, err
			}
			if len(validators) < 2 {
				return nil, fmt.Errorf("network should have at least two validators")
			}
			ok1, ok2 := false, false
			for _, val := range validators {
				if val.OperatorAddress == valFromAddr {
					args["validator-from"] = val
					ok1 = true
				} else {
					args["validator-to"] = val
					ok2 = true
				}
			}
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("cannot find a validator to redelegate")
			}

			args["validators"] = validators

			return args, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			senderAddr := args["sender-address"].(string)
			validatorFrom := args["validator-from"].(helpers.Validator)
			validatorTo := args["validator-to"].(helpers.Validator)
			validators := args["validators"].([]helpers.Validator)
			senderBalance := args["sender-balance"].(sdk.Coins)
			senderRewards := args["sender-rewards"].(sdk.Coins)

			newBalance, err := helpers.QueryBalanceOf(m, &node, senderAddr)
			if err != nil {
				return err
			}

			expectedBalance := senderBalance.Add(senderRewards[0]).Sub(sdk.NewCoin("aguru", sdk.NewInt(50400000000000000)))

			newValidators, err := helpers.QueryValidators(m, &node)
			if err != nil {
				return err
			}
			oldTotalPower := math.NewInt(0)
			newTotalPower := math.NewInt(0)
			for _, val := range validators {
				oldTotalPower = oldTotalPower.Add(val.Tokens)
			}
			for _, val := range newValidators {
				newTotalPower = newTotalPower.Add(val.Tokens)
			}

			// check bank module state changes
			if !expectedBalance.IsEqual(newBalance) {
				return fmt.Errorf("bank module state changes failed. expected: %s, got: %s", expectedBalance.String(), newBalance.String())
			}

			// check the staking module state changes
			delAmount, ok := sdk.NewIntFromString(redelegationAmount)
			if !ok {
				return fmt.Errorf("cannot convert %s to sdk.Int", redelegationAmount)
			}

			for _, val := range newValidators {
				if val.OperatorAddress == validatorFrom.OperatorAddress {
					if !val.Tokens.Add(delAmount).Equal(validatorFrom.Tokens) {
						return fmt.Errorf("staking module state changes failed. source validator tokens. expected: %s, got: %s", validatorFrom.Tokens.Sub(delAmount), val.Tokens)
					}
				}
				if val.OperatorAddress == validatorTo.OperatorAddress {
					if !val.Tokens.Sub(delAmount).Equal(validatorTo.Tokens) {
						return fmt.Errorf("staking module state changes failed. destination validator tokens. expected: %s, got: %s", validatorTo.Tokens.Add(delAmount), val.Tokens)
					}
				}
			}

			// check the gov module state changes
			if !oldTotalPower.Equal(newTotalPower) {
				return fmt.Errorf("gov module state changes failed")
			}

			// difficult to know the exact amount of distribution in dynamic network
			// // check the distribution module state changes
			// delegationRewards, err := helpers.QueryRewardsAll(m, &node, senderAddr)
			// if err != nil {
			// 	return err
			// }
			// delegationRewardsInt, _ := delegationRewards.TruncateDecimal()

			// if !senderRewards.IsAllGTE(delegationRewardsInt) || !senderRewards.IsEqual(delegationRewardsInt) {
			// 	return fmt.Errorf("distribution module state changes failed")
			// }

			return nil
		},
	})
}
