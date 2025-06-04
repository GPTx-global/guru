package oracle

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/GPTx-global/guru/x/oracle/keeper"
	"github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// InitGenesis new oracle genesis
func InitGenesis(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) {
	// Set genesis state
	params := data.Params
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(errorsmod.Wrapf(err, "error setting params"))
	}

	// Set moderator address
	moderatorAddress := data.ModeratorAddress
	if moderatorAddress == "" {
		panic(errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "%s: moderator address cannot be empty", types.ModuleName))
	}

	err = k.SetModeratorAddress(ctx, moderatorAddress)
	if err != nil {
		panic(errorsmod.Wrapf(err, "error setting moderator address"))
	}

	// Set oracle request documents
	oracleDocs := data.OracleRequestDocs
	for _, doc := range oracleDocs {
		err := doc.Validate()
		if err != nil {
			panic(errorsmod.Wrapf(err, "error validating oracle request doc"))
		}
		k.SetOracleRequestDoc(ctx, doc)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) types.GenesisState {
	var params types.Params
	var docs []types.OracleRequestDoc

	moderator_address := keeper.GetModeratorAddress(ctx)
	return types.NewGenesisState(params, docs, moderator_address)
}
