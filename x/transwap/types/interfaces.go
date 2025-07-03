package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cextypes "github.com/GPTx-global/guru/x/cex/types"
)

type CexKeeper interface {
	GetExchange(ctx sdk.Context, id math.Int) (*cextypes.Exchange, error)
	GetExchangePair(ctx sdk.Context, id math.Int, fromShortDenom, toShortDenom string) (*cextypes.Pair, error)
	SetAddressRequestCount(ctx sdk.Context, address string, count uint64) error
	GetAddressRequestCount(ctx sdk.Context, address string) uint64
	GetRatemeter(ctx sdk.Context) (*cextypes.Ratemeter, error)
}
