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
	"fmt"

	"github.com/GPTx-global/guru/x/feemarket/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/params"
)

var _ types.LegacyParams = &Params{}

// Parameter keys
var (
	ParamsKey                             = []byte("Params")
	ParamStoreKeyNoBaseFee                = []byte("NoBaseFee")
	ParamStoreKeyBaseFeeChangeDenominator = []byte("BaseFeeChangeDenominator")
	ParamStoreKeyElasticityMultiplier     = []byte("ElasticityMultiplier")
	ParamStoreKeyBaseFee                  = []byte("BaseFee")
	ParamStoreKeyEnableHeight             = []byte("EnableHeight")
	ParamStoreKeyMinGasPrice              = []byte("MinGasPrice")
	ParamStoreKeyMinGasMultiplier         = []byte("MinGasMultiplier")
)

var (
	// DefaultMinGasMultiplier is 0.5 or 50%
	DefaultMinGasMultiplier = sdk.NewDecWithPrec(50, 2)
	// DefaultMinGasPrice is 0 (i.e disabled)
	DefaultMinGasPrice  = sdk.ZeroDec()
	DefaultEnableHeight = int64(0)
	DefaultNoBaseFee    = false
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyNoBaseFee, &p.NoBaseFee, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyBaseFeeChangeDenominator, &p.BaseFeeChangeDenominator, validateBaseFeeChangeDenominator),
		paramtypes.NewParamSetPair(ParamStoreKeyElasticityMultiplier, &p.ElasticityMultiplier, validateElasticityMultiplier),
		paramtypes.NewParamSetPair(ParamStoreKeyBaseFee, &p.BaseFee, validateBaseFee),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableHeight, &p.EnableHeight, validateEnableHeight),
		paramtypes.NewParamSetPair(ParamStoreKeyMinGasPrice, &p.MinGasPrice, validateMinGasPrice),
		paramtypes.NewParamSetPair(ParamStoreKeyMinGasMultiplier, &p.MinGasMultiplier, validateMinGasPrice),
	}
}

// NewParams creates a new Params instance
func NewParams(
	noBaseFee bool,
	baseFeeChangeDenom,
	elasticityMultiplier uint32,
	baseFee uint64,
	enableHeight int64,
	minGasPrice sdk.Dec,
	minGasPriceMultiplier sdk.Dec,
) Params {
	return Params{
		NoBaseFee:                noBaseFee,
		BaseFeeChangeDenominator: baseFeeChangeDenom,
		ElasticityMultiplier:     elasticityMultiplier,
		BaseFee:                  sdkmath.NewIntFromUint64(baseFee),
		EnableHeight:             enableHeight,
		MinGasPrice:              minGasPrice,
		MinGasMultiplier:         minGasPriceMultiplier,
	}
}

// DefaultParams returns default evm parameters
func DefaultParams() Params {
	return Params{
		NoBaseFee:                DefaultNoBaseFee,
		BaseFeeChangeDenominator: params.DefaultBaseFeeChangeDenominator,
		ElasticityMultiplier:     params.DefaultElasticityMultiplier,
		BaseFee:                  sdkmath.NewIntFromUint64(params.InitialBaseFee),
		EnableHeight:             DefaultEnableHeight,
		MinGasPrice:              DefaultMinGasPrice,
		MinGasMultiplier:         DefaultMinGasMultiplier,
	}
}

// Validate performs basic validation on fee market parameters.
func (p Params) Validate() error {
	if p.BaseFeeChangeDenominator == 0 {
		return fmt.Errorf("base fee change denominator cannot be 0")
	}

	if p.BaseFee.IsNegative() {
		return fmt.Errorf("initial base fee cannot be negative: %s", p.BaseFee)
	}

	if p.EnableHeight < 0 {
		return fmt.Errorf("enable height cannot be negative: %d", p.EnableHeight)
	}

	if err := validateMinGasMultiplier(p.MinGasMultiplier); err != nil {
		return err
	}

	return validateMinGasPrice(p.MinGasPrice)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateBaseFeeChangeDenominator(i interface{}) error {
	value, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if value == 0 {
		return fmt.Errorf("base fee change denominator cannot be 0")
	}

	return nil
}

func validateElasticityMultiplier(i interface{}) error {
	_, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateBaseFee(i interface{}) error {
	value, ok := i.(sdkmath.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if value.IsNegative() {
		return fmt.Errorf("base fee cannot be negative")
	}

	return nil
}

func validateEnableHeight(i interface{}) error {
	value, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if value < 0 {
		return fmt.Errorf("enable height cannot be negative: %d", value)
	}

	return nil
}

func validateMinGasPrice(i interface{}) error {
	v, ok := i.(sdk.Dec)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("invalid parameter: nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %s", i)
	}

	return nil
}

func validateMinGasMultiplier(i interface{}) error {
	v, ok := i.(sdk.Dec)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("invalid parameter: nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("value cannot be greater than 1: %s", v)
	}
	return nil
}
