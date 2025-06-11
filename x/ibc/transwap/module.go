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

package transwap

import (
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/GPTx-global/guru/x/ibc/transwap/keeper"
	ibcexchange "github.com/cosmos/ibc-go/v6/modules/apps/exchange"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic embeds the IBC Transfer AppModuleBasic
type AppModuleBasic struct {
	*ibcexchange.AppModuleBasic
}

// AppModule represents the AppModule for this module
type AppModule struct {
	*ibcexchange.AppModule
	keeper keeper.Keeper
}

// NewAppModule creates a new 20-transfer module
func NewAppModule(k keeper.Keeper) AppModule {
	am := ibcexchange.NewAppModule(*k.Keeper)
	return AppModule{
		AppModule: &am,
		keeper:    k,
	}
}
