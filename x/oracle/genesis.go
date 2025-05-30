package oracle

import (
	"github.com/GPTx-global/guru/x/oracle/keeper"
	"github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis new oracle genesis
func InitGenesis(ctx sdk.Context, data types.GenesisState) {
	// keeper.SetModeratorAddress(ctx, data.ModeratorAddress)
	// keeper.SetCoinRates(ctx, data.CoinRates)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) types.GenesisState {
	var params types.Params
	var docs []types.RequestOracleDoc

	moderator_address := keeper.GetModeratorAddress(ctx)
	return types.NewGenesisState(params, docs, moderator_address)
}
