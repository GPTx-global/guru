package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cosmossdk_io_math "cosmossdk.io/math"
	cmd "github.com/GPTx-global/guru/tests/itest/cmd"
	"github.com/GPTx-global/guru/tests/itest/docker"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Validator struct {
	OperatorAddress string                      `protobuf:"bytes,1,opt,name=operator_address,json=operatorAddress,proto3" json:"operator_address,omitempty"`
	Tokens          cosmossdk_io_math.Int       `protobuf:"bytes,5,opt,name=tokens,proto3,customtype=cosmossdk.io/math.Int" json:"tokens"`
	Jailed          bool                        `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	DelegatorShares cosmossdk_io_math.LegacyDec `protobuf:"bytes,6,opt,name=delegator_shares,json=delegatorShares,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"delegator_shares"`
	Status          string                      `protobuf:"varint,4,opt,name=status,proto3,enum=cosmos.staking.v1beta1.BondStatus" json:"status,omitempty"`
}
type Validators struct {
	Validators []Validator `protobuf:"bytes,1,rep,name=validators,proto3" json:"validators"`
}

func GetValidatorAddresses(m *docker.Manager, node *NodeInfo) ([]string, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, stakingModuleName, "validators", "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res Validators
	if err := json.Unmarshal(outBuf.Bytes(), &res); err != nil {
		return nil, err
	}

	vals := []string{}
	for _, val := range res.Validators {
		vals = append(vals, val.OperatorAddress)
	}

	return vals, nil
}

func QueryValidators(m *docker.Manager, node *NodeInfo) ([]Validator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, stakingModuleName, "validators", "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res Validators
	if err := json.Unmarshal(outBuf.Bytes(), &res); err != nil {
		return nil, err
	}

	return res.Validators, nil
}

func QueryDelegations(m *docker.Manager, node *NodeInfo, addr string) ([]types.DelegationResponse, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return nil, err
	}

	queryExec, err := m.CreateQueryExec(conId, node.Branch, stakingModuleName, "delegations", addr, "--output", "json")
	if err != nil {
		return nil, err
	}
	outBuf, errBuf, err := m.RunExec(ctx, queryExec)
	if err != nil || errBuf.String() != "" {
		return nil, fmt.Errorf("execution failed: %s || err: %s", errBuf.String(), err)
	}
	var res types.QueryDelegatorDelegationsResponse
	codec.MustUnmarshalJSON([]byte(strings.TrimSpace(outBuf.String())), &res)

	return res.DelegationResponses, nil
}

func ExecCreateValidatorfunc(m *docker.Manager, node *NodeInfo, sender, moniker, amount, nodeId, pubkey string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conId, err := m.GetContainerId(node.Branch, node.RunOnIndex, node.RunOnFullnode)
	if err != nil {
		return err
	}

	cmd := cmd.CreateCreateValidatorCmd(sender, moniker, amount, pubkey, nodeId)
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
