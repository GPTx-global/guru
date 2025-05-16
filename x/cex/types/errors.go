package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrInvalidDenom         = errorsmod.Register(ModuleName, 2, "invalid denom")
	ErrWrongModerator       = errorsmod.Register(ModuleName, 3, "the operation is allowed only from moderator address")
	ErrWrongAdmin           = errorsmod.Register(ModuleName, 4, "the operation is allowed only from admin address")
	ErrInsufficientBalance  = errorsmod.Register(ModuleName, 5, "unsufficient balance")
	ErrInsufficientReserve  = errorsmod.Register(ModuleName, 6, "unsufficient reserve")
	ErrInvalidExchange      = errorsmod.Register(ModuleName, 7, "invalid exchange attribute")
	ErrInvalidExchangeId    = errorsmod.Register(ModuleName, 8, "invalid exchange id")
	ErrInvalidAdmin         = errorsmod.Register(ModuleName, 9, "invalid admin address")
	ErrInvalidReserve       = errorsmod.Register(ModuleName, 10, "invalid reserve address")
	ErrInvalidCoin          = errorsmod.Register(ModuleName, 11, "invalid coin info")
	ErrInvalidAttribute     = errorsmod.Register(ModuleName, 12, "invalid attribute")
	ErrUnableToFetch        = errorsmod.Register(ModuleName, 13, "unable to fetch")
	ErrInvalidJsonFile      = errorsmod.Register(ModuleName, 14, "invalid json file")
	ErrConstantKey          = errorsmod.Register(ModuleName, 15, "value cannot be updated")
	ErrRequiredKey          = errorsmod.Register(ModuleName, 16, "required key not provided")
	ErrKeyNotFound          = errorsmod.Register(ModuleName, 17, "attribute with the given key not found")
	ErrInvalidExchangeCoins = errorsmod.Register(ModuleName, 18, "invalid exchange coins")
)
