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
	for _, exchange := range data.Exchanges {
		keeper.AddNewExchange(ctx, &exchange)
	}
	for _, admin := range data.Admins {
		keeper.AddNewAdmin(ctx, admin)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) types.GenesisState {
	moderator_address := keeper.GetModeratorAddress(ctx)
	exchanges, _, err := keeper.GetPaginatedExchanges(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		panic(fmt.Errorf("unable to fetch exchanges %v", err))
	}
	admins, _, err := keeper.GetPaginatedAdmins(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		panic(fmt.Errorf("unable to fetch admins %v", err))
	}
	return types.NewGenesisState(moderator_address, exchanges, admins)
}
