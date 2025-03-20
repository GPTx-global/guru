package itest

import (
	"context"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
)

func addMsgSendCases() {
	helpers.AddTestCase(&helpers.TestCase{
		// Reset: false,
		Node: helpers.NodeInfo{
			Branch: "v1.0.6",
			Vals:   1,
			Fulls:  0,
		},
		Module: "bank",
		Name:   "should pass - send",
		Cmd: func(args map[string]interface{}) []string {
			return cmd.CreateSendCmd("dev0", "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h", "10aguru", "630000000000aguru", "30000")
		},
		ExpPass: true,
		ExpErr:  "",
		CheckCases: []helpers.CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h"},
				Expected: "balances:\n- amount: \"10\"\n  denom: aguru\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		Malleate_pre: func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error) {
			return make(map[string]interface{}), nil
		},
		Malleate_post: func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error {
			return nil
		},
	})

}
