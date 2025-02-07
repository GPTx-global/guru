package e2e

import (
	bankcmd "github.com/GPTx-global/guru/tests/e2e/cmd/bank"
)

func init() {

	AddTestCase(&TestCase{
		Reset:   false,
		Branch:  "main",
		Module:  "bank",
		Name:    "should pass - send",
		Cmd:     bankcmd.CreateSendCmd("mykey", "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h", "10aguru", "630000000000aguru", "30000"),
		ExpPass: true,
		ExpErr:  "",
		PassCheck: []CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h"},
				Expected: "balances:\n- amount: \"10\"\n  denom: aguru\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		FailCheck: []CheckCase{},
		PassCheckFunc: func() error {
			return nil
		},
		FailCheckFunc: func() error {
			return nil
		},
	})

	AddTestCase(&TestCase{
		Reset:     true,
		Branch:    "main",
		Module:    "bank",
		Name:      "should fail - send",
		Cmd:       bankcmd.CreateSendCmd("mykey1", "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h", "10aguru", "630000000000aguru", "30000"),
		ExpPass:   false,
		ExpErr:    "",
		PassCheck: []CheckCase{},
		FailCheck: []CheckCase{
			{
				Module:   "bank",
				Query:    "balances",
				Args:     []string{"guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h"},
				Expected: "balances: []\npagination:\n  next_key: null\n  total: \"0\"\n",
			},
		},
		PassCheckFunc: func() error {
			return nil
		},
		FailCheckFunc: func() error {
			return nil
		},
	})
}
