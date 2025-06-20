package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrInvalidDenom        = errorsmod.Register(ModuleName, 2, "invalid denom")
	ErrWrongModerator      = errorsmod.Register(ModuleName, 3, "the operation is allowed only from moderator address")
	ErrWrongAdmin          = errorsmod.Register(ModuleName, 4, "the operation is allowed only from admin address")
	ErrInsufficientBalance = errorsmod.Register(ModuleName, 5, "unsufficient balance")
	ErrInsufficientReserve = errorsmod.Register(ModuleName, 6, "unsufficient reserve")
	ErrInvalidExchange     = errorsmod.Register(ModuleName, 7, "invalid exchange attribute")
	ErrInvalidRatemeter    = errorsmod.Register(ModuleName, 8, "invalid ratemeter")
	ErrInvalidAdmin        = errorsmod.Register(ModuleName, 9, "invalid admin address")
	ErrInvalidAttribute    = errorsmod.Register(ModuleName, 10, "invalid attribute")
	ErrNotFound            = errorsmod.Register(ModuleName, 11, "not found")
	ErrInvalidJsonFile     = errorsmod.Register(ModuleName, 12, "invalid json file")
	ErrConstantKey         = errorsmod.Register(ModuleName, 13, "value cannot be updated")
	ErrRequiredKey         = errorsmod.Register(ModuleName, 14, "required key not provided")
)
