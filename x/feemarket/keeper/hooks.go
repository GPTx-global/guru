package keeper

import (
	"github.com/GPTx-global/guru/x/feemarket/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeforeOracleStart: noop, We don't need to do anything here
func (k Keeper) BeforeOracleStart(_ sdk.Context, _ oracletypes.DataSet) {
}

// AfterOracleEnd mints and allocates coins at the end of each oracle end
func (k Keeper) AfterOracleEnd(ctx sdk.Context, dataSet oracletypes.DataSet) {
	logger := ctx.Logger()
	logger.Info("AfterOracleEnd hook triggered", "dataSet", dataSet)

	params := k.GetParams(ctx)
	minGasPriceRate := params.MinGasPriceRate

	if minGasPriceRate.IsZero() {
		return
	}

	newMinGasPrice := minGasPriceRate.Quo(sdk.MustNewDecFromStr(dataSet.RawData))
	params.MinGasPrice = newMinGasPrice

	k.SetParams(ctx, params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChangeMinGasPrice,
			sdk.NewAttribute(types.AttributeKeyMinGasPrice, newMinGasPrice.String()),
		),
	)
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for incentives keeper
type Hooks struct {
	k Keeper
}

var _ oracletypes.OracleHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// oracle hooks
func (h Hooks) BeforeOracleStart(ctx sdk.Context, dataSet oracletypes.DataSet) {
	h.k.BeforeOracleStart(ctx, dataSet)
}

func (h Hooks) AfterOracleEnd(ctx sdk.Context, dataSet oracletypes.DataSet) {
	h.k.AfterOracleEnd(ctx, dataSet)
}
