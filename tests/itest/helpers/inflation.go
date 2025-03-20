package helpers

import (
	"context"
	"fmt"

	"github.com/GPTx-global/guru/tests/itest/docker"
)

func QueryCirculatingSupply(m *docker.Manager, node *NodeInfo) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return "", err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, inflationModuleName, "circulating-supply")
	if err != nil {
		return "", err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return "", fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}

	return RemoveWhiteSpace(outBuf.String()), nil
}
