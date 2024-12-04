// This accounts file contains the accounts and corresponding token allocation for
// accounts which participated in the Olympus Mons Testnet (November 2021) through
// completion of the Mars Meteor Missions. The token allocation is in aguru.

// 7.5% of the genesis allocation, totaling to ~7.5M Evmos tokens, was set aside for
// participants of the incentivized testnet, of which ~5.6M is claimed here. The
// remaining funds will be sent to the community pool.

// The participants of the testnet will immediately stake their tokens to a set
// of chosen validators, also included in this file.

package v11

// Allocations are rewards by participant address, along with the randomized validator to stake to.
var Allocations = [5][3]string{
	{
		"guru10jmp6sgh4cc6zt3e8gw05wavvejgr5pwggsdaj",
		"72232472320000001048576",
		"guruvaloper10jmp6sgh4cc6zt3e8gw05wavvejgr5pwrdtkut",
	},
	{
		"guru1cml96vmptgw99syqrrz8az79xer2pcgpawsch9",
		"48985239849999997599744",
		"guruvaloper1cml96vmptgw99syqrrz8az79xer2pcgpkttrku",
	},
	{
		"guru1jcltmuhplrdcwp7stlr4hlhlhgd4htqhtx0smk",
		"46494464939999998509056",
		"guruvaloper1jcltmuhplrdcwp7stlr4hlhlhgd4htqhqr5t60",
	},
	{
		"guru1gzsvk8rruqn2sx64acfsskrwy8hvrmaf6dvhj3",
		"39852398519999998722048",
		"guruvaloper1gzsvk8rruqn2sx64acfsskrwy8hvrmaf3ghvng",
	},
	{
		"guru1fx944mzagwdhx0wz7k9tfztc8g3lkfk6ecee3f",
		"37361623620000000507904",
		"guruvaloper1fx944mzagwdhx0wz7k9tfztc8g3lkfk6jazzss",
	},
}
