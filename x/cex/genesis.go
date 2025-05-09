package cex

import (
	"fmt"

	"github.com/GPTx-global/guru/x/cex/keeper"
	"github.com/GPTx-global/guru/x/cex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// InitGenesis new cex genesis
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, bk types.BankKeeper, data types.GenesisState) {
	keeper.SetModeratorAddress(ctx, data.ModeratorAddress)
	keeper.SetCoinPairs(ctx, data.CoinPairs)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) types.GenesisState {
	moderator_address := keeper.GetModeratorAddress(ctx)
	coin_pairs, _, err := keeper.GetPaginatedCoinPairs(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		panic(fmt.Errorf("unable to fetch coin pairs %v", err))
	}
	return types.NewGenesisState(moderator_address, coin_pairs)
}
