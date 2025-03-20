package helpers

import (
	"context"
	"fmt"
	"strings"

	// cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func QueryBaseAddress(m *docker.Manager, node *NodeInfo) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return "", err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "base-address", "--output", "json")
	if err != nil {
		return "", err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return "", fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryBaseAddressResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.BaseAddress, nil
}

func QueryModeratorAddress(m *docker.Manager, node *NodeInfo) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return "", err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "moderator-address", "--output", "json")
	if err != nil {
		return "", err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return "", fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryModeratorResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.ModeratorAddress, nil
}

func QueryCommunityPool(m *docker.Manager, node *NodeInfo) (sdk.DecCoins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "community-pool", "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryCommunityPoolResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Pool, nil
}

func QueryValidatorOutstandingRewards(m *docker.Manager, node *NodeInfo, validator string) (sdk.DecCoins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "validator-outstanding-rewards", validator, "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.ValidatorOutstandingRewards
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Rewards, nil
}

func QueryRewards(m *docker.Manager, node *NodeInfo, delegator, validator string) (sdk.DecCoins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "rewards", delegator, validator, "--output", "json")
	if err != nil {
		return nil, err
	}

	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryDelegationRewardsResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Rewards, nil
}

func QueryRewardsAll(m *docker.Manager, node *NodeInfo, delegator string) (sdk.DecCoins, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, distributionModuleName, "rewards", delegator, "--output", "json")
	if err != nil {
		return nil, err
	}

	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("query failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryDelegationTotalRewardsResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.Total, nil
}
