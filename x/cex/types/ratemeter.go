package types

import (
	time "time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ValidateRatemeter(ratemeter *Ratemeter) error {
	if ratemeter.RequestCountLimit.IsNil() {
		return errorsmod.Wrapf(ErrInvalidRatemeter, "request count limit is nil")
	}

	if ratemeter.RequestCountLimit.IsZero() {
		return errorsmod.Wrapf(ErrInvalidRatemeter, "request count limit is zero")
	}

	if ratemeter.RequestCountLimit.IsNegative() {
		return errorsmod.Wrapf(ErrInvalidRatemeter, "request count limit is negative")
	}

	if ratemeter.RequestPeriod <= 0 {
		return errorsmod.Wrapf(ErrInvalidRatemeter, "request period must be positive")
	}

	return nil
}

func DefaultRatemeter() Ratemeter {
	return Ratemeter{
		RequestCountLimit: sdk.NewInt(5),
		RequestPeriod:     time.Hour * 24,
	}
}
