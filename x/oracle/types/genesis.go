package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params, docs []RequestOracleDoc, moderatorAddress string) GenesisState {
	return GenesisState{
		Params:            params,
		RequestOracleDocs: docs,
		ModeratorAddress:  moderatorAddress,
	}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		RequestOracleDocs: []RequestOracleDoc{},
		ModeratorAddress:  "",
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Validate each request oracle doc
	for _, doc := range gs.RequestOracleDocs {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("invalid request oracle doc: %w", err)
		}
	}

	// Validate moderator address if provided
	if gs.ModeratorAddress != "" {
		if _, err := sdk.AccAddressFromBech32(gs.ModeratorAddress); err != nil {
			return fmt.Errorf("invalid moderator address: %w", err)
		}
	}

	return nil
}
