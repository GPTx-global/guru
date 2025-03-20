package cmd

import "fmt"

func CreateUnjailCmd(sender string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"slashing",
		"unjail",
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--gas-prices=%s", "630000000000aguru"),
		fmt.Sprintf("--gas=%s", "30000"),
		"-y",
	}
	return cmd
}
