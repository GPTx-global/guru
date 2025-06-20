// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	exchangetypes "github.com/cosmos/ibc-go/v6/modules/apps/exchange/types"

	cextypes "github.com/GPTx-global/guru/x/cex/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	exchangetypes.AccountKeeper
	GetAccount(sdk.Context, sdk.AccAddress) authtypes.AccountI
}

// BankKeeper defines the expected interface needed to check balances and send coins.
type BankKeeper interface {
	exchangetypes.BankKeeper
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type CexKeeper interface {
	GetExchange(ctx sdk.Context, id math.Int) (*cextypes.Exchange, error)
	GetExchangePair(ctx sdk.Context, id math.Int, fromShortDenom, toShortDenom string) (*cextypes.Pair, error)
	SetAddressRequestCount(ctx sdk.Context, address string, count uint64) error
	GetAddressRequestCount(ctx sdk.Context, address string) uint64
	GetRatemeter(ctx sdk.Context) (*cextypes.Ratemeter, error)
	// GetExchangeAttribute(ctx sdk.Context, id math.Int, key string) (cextypes.Attribute, error)
}
