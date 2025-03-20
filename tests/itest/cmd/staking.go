package cmd

import "fmt"

func CreateCreateValidatorCmd(sender, moniker, amount, pubkey, nodeId string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"staking",
		"create-validator",
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--moniker=%s", moniker),
		fmt.Sprintf("--amount=%s", amount),
		fmt.Sprintf("--pubkey=%s", pubkey),
		fmt.Sprintf("--node-id=%s", nodeId),
		fmt.Sprintf("--commission-max-change-rate=%s", "0.01"),
		fmt.Sprintf("--commission-max-rate=%s", "0.2"),
		fmt.Sprintf("--commission-rate=%s", "0.1"),
		fmt.Sprintf("--min-self-delegation=%s", "1"),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "80000"),
		"-y",
	}
	return cmd
}

func CreateDelegateCmd(sender, validator, amount string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"staking",
		"delegate",
		validator,
		amount,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "80000"),
		"-y",
	}
	return cmd
}

func CreateRedelegateCmd(sender, validatorFrom, validatorTo, amount string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"staking",
		"redelegate",
		validatorFrom,
		validatorTo,
		amount,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "80000"),
		"-y",
	}
	return cmd
}

func CreateUnbondCmd(sender, validator, amount string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"staking",
		"unbond",
		validator,
		amount,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "80000"),
		"-y",
	}
	return cmd
}
