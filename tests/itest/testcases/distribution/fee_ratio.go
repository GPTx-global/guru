package itest

import (
	"context"
	"fmt"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func addFeeRatioDistributionCases() {
	var branch = "v1.0.6"
	var module = "distribution"

	var node = helpers.NodeInfo{Branch: branch}

	helpers.AddTestCase(&helpers.TestCase{
		Node: helpers.NodeInfo{
			Branch: branch,
			Vals:   1,
			Fulls:  0,
		},
		Module: module,
		Name:   "check the correct fee distribution",
		Cmd: func(args map[string]interface{}) []string {
			return cmd.CreateSendCmd("dev0", "guru146q0lys3vrstcl2yckytayupscy52twmvcpa9v", "10aguru", "630000000000aguru", "30000")
		},
		ExpPass: true,
		ExpErr:  "",
		CheckCases: []helpers.CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru146q0lys3vrstcl2yckytayupscy52twmvcpa9v"},
				Expected: "balances:\n- amount: \"10\"\n  denom: aguru\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			// conId, err := m.GetValidatorConId(branch, 0)
			// if err != nil {
			// 	return nil, err
			// }
			// m.WaitForNextBlock(ctx, conId)

			initialValues := make(map[string]interface{})

			baseAddr, err := helpers.QueryBaseAddress(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			initialValues["base-address"] = baseAddr

			baseBalance, err := helpers.QueryBalanceOf(m, &node, baseAddr)
			if err != nil {
				return nil, err
			}
			initialValues["base-balance"] = baseBalance

			totalSupply, err := helpers.QueryTotalSupply(m, &node)
			if err != nil {
				return nil, err
			}
			initialValues["total-supply"] = totalSupply

			validators, err := helpers.GetValidatorAddresses(m, &node)
			if err != nil {
				return nil, err
			}
			initialValues["validator-list"] = validators

			var stakingRewards = sdk.DecCoins{}
			for _, val := range validators {
				reward, err := helpers.QueryValidatorOutstandingRewards(m, &helpers.NodeInfo{Branch: branch}, val)
				if err != nil {
					return nil, err
				}
				stakingRewards = stakingRewards.Add(reward...)
			}

			pool, err := helpers.QueryCommunityPool(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return nil, err
			}
			stakingRewards = stakingRewards.Add(pool...)
			initialValues["staking-rewards"] = stakingRewards

			return initialValues, nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			// load the initial values
			baseAddress := args["base-address"].(string)
			initialBaseBalance := args["base-balance"].(sdk.Coins)
			initialTotalSupply := args["total-supply"].(sdk.Coins)
			validatorList := args["validator-list"].([]string)
			initialStakingRewards := args["staking-rewards"].(sdk.DecCoins)

			// query the new values
			baseBalance, err := helpers.QueryBalanceOf(m, &node, baseAddress)
			if err != nil {
				return err
			}
			totalSupply, err := helpers.QueryTotalSupply(m, &node)
			if err != nil {
				return err
			}

			var stakingRewards = sdk.DecCoins{}
			for _, val := range validatorList {
				reward, err := helpers.QueryValidatorOutstandingRewards(m, &helpers.NodeInfo{Branch: branch}, val)
				if err != nil {
					return err
				}
				stakingRewards = stakingRewards.Add(reward...)
			}

			pool, err := helpers.QueryCommunityPool(m, &helpers.NodeInfo{Branch: branch})
			if err != nil {
				return err
			}
			stakingRewards = stakingRewards.Add(pool...)

			if !initialBaseBalance.Add(sdk.NewCoin("aguru", sdk.NewInt(6299999999999999))).IsEqual(baseBalance) {
				return fmt.Errorf("base rewards are not correct. expected: %s, got: %s", initialBaseBalance.String(), baseBalance.String())
			}

			if !initialTotalSupply.Sub(sdk.NewCoin("aguru", sdk.NewInt(6299999999999999))).IsEqual(totalSupply) {
				return fmt.Errorf("burn amount not correct. expected: %s, got: %s", initialTotalSupply.String(), totalSupply.String())
			}

			if !initialStakingRewards.Add(sdk.NewDecCoin("aguru", sdk.NewInt(6300000000000002))).IsEqual(stakingRewards) {
				return fmt.Errorf("staking rewards is not correct. expected: %s, got: %s", initialStakingRewards.String(), stakingRewards.String())
			}

			return nil
		},
	})

}
