package cmd

import "fmt"

func CreateSendCmd(sender, receiver, amount, gasPrice, gas string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"bank",
		"send",
		sender,
		receiver,
		amount,
		fmt.Sprintf("--gas-prices=%s", gasPrice),
		fmt.Sprintf("--gas=%s", gas),
		"-y",
	}
	return cmd
}

func CreateMultiSendCmd(sender, amount, gasPrice, gas string, receivers []string) []string {
	cmd := []string{
		"gurud",
		"tx",
		"bank",
		"multi-send",
		sender,
	}
	cmd = append(cmd, receivers...)
	cmd = append(cmd, amount, fmt.Sprintf("--gas-prices=%s", gasPrice), fmt.Sprintf("--gas=%s", gas), "-y")
	return cmd
}
