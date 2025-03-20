package helpers

import (
	"strings"

	amino "github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

const (
	bankModuleName         = "bank"
	distributionModuleName = "distribution"
	stakingModuleName      = "staking"
	inflationModuleName    = "inflation"
)

var (
	codec = amino.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func RemoveWhiteSpace(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\n", ""), "\r", "")
}
