package cmd

import (
	cmd "github.com/GPTx-global/guru/tests/e2e/cmd"
)

func addMsgSendCases() {
	cmd.AddTestCase(&cmd.TestCase{
		Reset:   false,
		Branch:  "main",
		Module:  "bank",
		Name:    "should pass - send",
		Cmd:     CreateSendCmd("mykey", "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h", "10aguru", "630000000000aguru", "30000"),
		ExpPass: true,
		ExpErr:  "",
		CheckCases: []cmd.CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h"},
				Expected: "balances:\n- amount: \"10\"\n  denom: aguru\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		Malleate_pre: func() error {
			return nil
		},
		Malleate_post: func() error {
			return nil
		},
	})

	cmd.AddTestCase(&cmd.TestCase{
		Reset:   true,
		Branch:  "main",
		Module:  "bank",
		Name:    "should fail - send",
		Cmd:     CreateSendCmd("mykey1", "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h", "10aguru", "630000000000aguru", "30000"),
		ExpPass: false,
		ExpErr:  "",
		CheckCases: []cmd.CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h"},
				Expected: "balances: []\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		Malleate_pre: func() error {
			return nil
		},
		Malleate_post: func() error {
			return nil
		},
	})
}
