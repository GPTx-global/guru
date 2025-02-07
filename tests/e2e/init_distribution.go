package e2e

import (
	bankcmd "github.com/GPTx-global/guru/tests/e2e/cmd/bank"
)

func init() {
	AddTestCase(&TestCase{
		Reset:   true,
		Branch:  "main",
		Module:  "distribution",
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
			{
				Module:   "bank",
				Query:    "total",
				Args:     []string{},
				Expected: "pagination:\n  next_key: null\n  total: \"0\"\nsupply:\n- amount: \"100000000000000000000010000\"\n  denom: aguru\n",
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

}
