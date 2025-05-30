package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns default oracle module parameters
func DefaultParams() Params {
	return Params{
		EnableOracle:          true,
		SubmitWindow:          3600, // 1 hour in seconds
		MinSubmitPerWindow:    sdk.NewDec(1),
		SlashFractionDowntime: sdk.NewDecWithPrec(1, 2), // 1%
	}
}

// Validate performs basic validation on oracle parameters
func (p Params) Validate() error {
	if p.SubmitWindow == 0 {
		return fmt.Errorf("submit window cannot be zero")
	}

	if p.MinSubmitPerWindow.IsNegative() {
		return fmt.Errorf("min submit per window cannot be negative")
	}

	if p.SlashFractionDowntime.IsNegative() {
		return fmt.Errorf("slash fraction downtime cannot be negative")
	}

	return nil
}
