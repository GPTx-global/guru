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

package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/exchange/keeper"
	exchangetypes "github.com/cosmos/ibc-go/v6/modules/apps/exchange/types"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"

	"github.com/GPTx-global/guru/x/ibc/transwap/types"
)

// Keeper defines the modified IBC transfer keeper that embeds the original one.
// It also contains the bank keeper and the erc20 keeper to support ERC20 tokens
// to be sent via IBC.
type Keeper struct {
	*keeper.Keeper
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
	cexKeeper     types.CexKeeper
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramSpace paramtypes.Subspace,

	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper exchangetypes.ChannelKeeper,
	portKeeper exchangetypes.PortKeeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
	cexKeeper types.CexKeeper,
) Keeper {
	// create the original IBC transfer keeper for embedding
	transferKeeper := keeper.NewKeeper(
		cdc, storeKey, paramSpace,
		ics4Wrapper, channelKeeper, portKeeper,
		accountKeeper, bankKeeper, scopedKeeper,
	)

	return Keeper{
		Keeper:        &transferKeeper,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		cexKeeper:     cexKeeper,
	}
}
