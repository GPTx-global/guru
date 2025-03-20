package cmd

import "fmt"

func CreateChangeRatioCmd(sender, staking_rewards, base, burn string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"distribution",
		"change-ratio",
		staking_rewards,
		base,
		burn,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "40000"),
		"-y",
	}
	return cmd
}

func CreateChangeModeratorCmd(sender, new_moderator string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"distribution",
		"change-moderator",
		new_moderator,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "40000"),
		"-y",
	}
	return cmd
}

func CreateChangeBaseAddressCmd(sender, new_base_address string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"distribution",
		"change-base-address",
		new_base_address,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "40000"),
		"-y",
	}
	return cmd
}
