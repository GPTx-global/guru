package helpers

import (
	"context"
	"fmt"
	"strings"

	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

func QueryBalanceOf(m *docker.Manager, node *NodeInfo, addr string) (sdk.Coins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, bankModuleName, "balances", addr, "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryAllBalancesResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Balances, nil
}

func QueryTotalSupply(m *docker.Manager, node *NodeInfo) (sdk.Coins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, bankModuleName, "total", "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryTotalSupplyResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Supply, nil
}

func ExecSend(m *docker.Manager, node *NodeInfo, from, to, amount string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return err
	}

	cmd := cmd.CreateSendCmd(from, to, amount, "630000000000aguru", "30000")
	exec, err := m.CreateExec(cmd, conId)
	if err != nil {
		return err
	}

	outBuf, errBuf, err := m.RunExec(ctx, exec)
	if err != nil || errBuf.String() != "" {
		return fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}

	if !strings.Contains(outBuf.String(), "code: 0") {
		return fmt.Errorf("tx returned non code 0:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String())
	}

	return nil
}

func ExecBankMultiSend(m *docker.Manager, node *NodeInfo, from, amount string, receivers []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return err
	}

	cmd := cmd.CreateMultiSendCmd(from, amount, "630000000000aguru", "1000000", receivers)
	exec, err := m.CreateExec(cmd, conId)
	if err != nil {
		return err
	}

	outBuf, errBuf, err := m.RunExec(ctx, exec)
	if err != nil || errBuf.String() != "" {
		return fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}

	if !strings.Contains(outBuf.String(), "code: 0") {
		return fmt.Errorf("tx returned non code 0:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String())
	}

	return nil
}
