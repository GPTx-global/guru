package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params, docs []OracleRequestDoc, moderatorAddress string, predefinedOracles []PredefinedOracle) GenesisState {
	return GenesisState{
		Params:            params,
		OracleRequestDocs: docs,
		ModeratorAddress:  moderatorAddress,
		PredefinedOracles: predefinedOracles,
	}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		OracleRequestDocs: []OracleRequestDoc{},
		ModeratorAddress:  "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft", // This is a test address
		PredefinedOracles: []PredefinedOracle{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Validate each request oracle doc
	for _, doc := range gs.OracleRequestDocs {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("invalid request oracle doc: %w", err)
		}
	}

	// Validate params
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate moderator address if provided
	if gs.ModeratorAddress != "" {
		if _, err := sdk.AccAddressFromBech32(gs.ModeratorAddress); err != nil {
			return fmt.Errorf("invalid moderator address: %w", err)
		}
	}

	return nil
}
