package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrInvalidPairDenom    = errorsmod.Register(ModuleName, 2, "invalid pair denom")
	ErrWrongModerator      = errorsmod.Register(ModuleName, 3, "the operation is allowed only from moderator address")
	ErrWrongAdmin          = errorsmod.Register(ModuleName, 4, "the operation is allowed only from admin address")
	ErrInsufficientBalance = errorsmod.Register(ModuleName, 5, "unsufficient balance")
	ErrInsufficientReserve = errorsmod.Register(ModuleName, 6, "unsufficient reserve")
)
