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
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func newUint64(val uint64) *uint64 { return &val }

// EthereumConfig returns an Ethereum ChainConfig for EVM state transitions.
// All the negative or nil values are converted to nil
func (cc ChainConfig) EthereumConfig(chainID *big.Int) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 chainID,
		HomesteadBlock:          getBlockValue(cc.HomesteadBlock),
		DAOForkBlock:            getBlockValue(cc.DAOForkBlock),
		DAOForkSupport:          cc.DAOForkSupport,
		EIP150Block:             getBlockValue(cc.EIP150Block),
		EIP155Block:             getBlockValue(cc.EIP155Block),
		EIP158Block:             getBlockValue(cc.EIP158Block),
		ByzantiumBlock:          getBlockValue(cc.ByzantiumBlock),
		ConstantinopleBlock:     getBlockValue(cc.ConstantinopleBlock),
		PetersburgBlock:         getBlockValue(cc.PetersburgBlock),
		IstanbulBlock:           getBlockValue(cc.IstanbulBlock),
		MuirGlacierBlock:        getBlockValue(cc.MuirGlacierBlock),
		BerlinBlock:             getBlockValue(cc.BerlinBlock),
		LondonBlock:             getBlockValue(cc.LondonBlock),
		ArrowGlacierBlock:       getBlockValue(cc.ArrowGlacierBlock),
		GrayGlacierBlock:        getBlockValue(cc.GrayGlacierBlock),
		MergeNetsplitBlock:      getBlockValue(cc.MergeNetsplitBlock),
		ShanghaiTime:            getTimeValue(cc.ShanghaiTime),
		CancunTime:              getTimeValue(cc.CancunTime),
		TerminalTotalDifficulty: nil,
		Ethash:                  nil,
		Clique:                  nil,
	}
}

// DefaultChainConfig returns default evm parameters.
func DefaultChainConfig() ChainConfig {
	homesteadBlock := sdk.ZeroInt()
	daoForkBlock := sdk.ZeroInt()
	eip150Block := sdk.ZeroInt()
	eip155Block := sdk.ZeroInt()
	eip158Block := sdk.ZeroInt()
	byzantiumBlock := sdk.ZeroInt()
	constantinopleBlock := sdk.ZeroInt()
	petersburgBlock := sdk.ZeroInt()
	istanbulBlock := sdk.ZeroInt()
	muirGlacierBlock := sdk.ZeroInt()
	berlinBlock := sdk.ZeroInt()
	londonBlock := sdk.ZeroInt()
	arrowGlacierBlock := sdk.ZeroInt()
	grayGlacierBlock := sdk.ZeroInt()
	mergeNetsplitBlock := sdk.ZeroInt()
	ShanghaiTime := sdk.NewUint(0)
	CancunTime := sdk.NewUint(0)

	return ChainConfig{
		HomesteadBlock:      &homesteadBlock,
		DAOForkBlock:        &daoForkBlock,
		DAOForkSupport:      true,
		EIP150Block:         &eip150Block,
		EIP150Hash:          common.Hash{}.String(),
		EIP155Block:         &eip155Block,
		EIP158Block:         &eip158Block,
		ByzantiumBlock:      &byzantiumBlock,
		ConstantinopleBlock: &constantinopleBlock,
		PetersburgBlock:     &petersburgBlock,
		IstanbulBlock:       &istanbulBlock,
		MuirGlacierBlock:    &muirGlacierBlock,
		BerlinBlock:         &berlinBlock,
		LondonBlock:         &londonBlock,
		ArrowGlacierBlock:   &arrowGlacierBlock,
		GrayGlacierBlock:    &grayGlacierBlock,
		MergeNetsplitBlock:  &mergeNetsplitBlock,
		ShanghaiTime:        &ShanghaiTime,
		CancunTime:          &CancunTime,
	}
}

func getBlockValue(block *sdkmath.Int) *big.Int {
	if block == nil || block.IsNegative() {
		return nil
	}

	return block.BigInt()
}

func getTimeValue(time *sdkmath.Uint) *uint64 {
	if time == nil {
		return nil
	}

	return newUint64(time.Uint64())
}

// Validate performs a basic validation of the ChainConfig params. The function will return an error
// if any of the block values is uninitialized (i.e nil) or if the EIP150Hash is an invalid hash.
func (cc ChainConfig) Validate() error {
	if err := validateBlock(cc.HomesteadBlock); err != nil {
		return errorsmod.Wrap(err, "homesteadBlock")
	}
	if err := validateBlock(cc.DAOForkBlock); err != nil {
		return errorsmod.Wrap(err, "daoForkBlock")
	}
	if err := validateBlock(cc.EIP150Block); err != nil {
		return errorsmod.Wrap(err, "eip150Block")
	}
	if err := validateHash(cc.EIP150Hash); err != nil {
		return err
	}
	if err := validateBlock(cc.EIP155Block); err != nil {
		return errorsmod.Wrap(err, "eip155Block")
	}
	if err := validateBlock(cc.EIP158Block); err != nil {
		return errorsmod.Wrap(err, "eip158Block")
	}
	if err := validateBlock(cc.ByzantiumBlock); err != nil {
		return errorsmod.Wrap(err, "byzantiumBlock")
	}
	if err := validateBlock(cc.ConstantinopleBlock); err != nil {
		return errorsmod.Wrap(err, "constantinopleBlock")
	}
	if err := validateBlock(cc.PetersburgBlock); err != nil {
		return errorsmod.Wrap(err, "petersburgBlock")
	}
	if err := validateBlock(cc.IstanbulBlock); err != nil {
		return errorsmod.Wrap(err, "istanbulBlock")
	}
	if err := validateBlock(cc.MuirGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "muirGlacierBlock")
	}
	if err := validateBlock(cc.BerlinBlock); err != nil {
		return errorsmod.Wrap(err, "berlinBlock")
	}
	if err := validateBlock(cc.LondonBlock); err != nil {
		return errorsmod.Wrap(err, "londonBlock")
	}
	if err := validateBlock(cc.ArrowGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "arrowGlacierBlock")
	}
	if err := validateBlock(cc.GrayGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "GrayGlacierBlock")
	}
	if err := validateBlock(cc.MergeNetsplitBlock); err != nil {
		return errorsmod.Wrap(err, "MergeNetsplitBlock")
	}
	if err := validateTime(cc.ShanghaiTime); err != nil {
		return errorsmod.Wrap(err, "ShanghaiTime")
	}
	if err := validateTime(cc.CancunTime); err != nil {
		return errorsmod.Wrap(err, "CancunTime")
	}
	// NOTE: chain ID is not needed to check config order
	if err := cc.EthereumConfig(nil).CheckConfigForkOrder(); err != nil {
		return errorsmod.Wrap(err, "invalid config fork order")
	}
	return nil
}

func validateHash(hex string) error {
	if hex != "" && strings.TrimSpace(hex) == "" {
		return errorsmod.Wrap(ErrInvalidChainConfig, "hash cannot be blank")
	}

	return nil
}

func validateBlock(block *sdkmath.Int) error {
	// nil value means that the fork has not yet been applied
	if block == nil {
		return nil
	}

	if block.IsNegative() {
		return errorsmod.Wrapf(
			ErrInvalidChainConfig, "block value cannot be negative: %s", block,
		)
	}

	return nil
}

func validateTime(time *sdkmath.Uint) error {
	// nil value means that the fork has not yet been applied
	if time == nil {
		return nil
	}

	return nil
}
