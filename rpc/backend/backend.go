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
package backend

import (
	"context"
	"math/big"
	"time"

	rpctypes "github.com/GPTx-global/guru/rpc/types"
	"github.com/GPTx-global/guru/server/config"
	evmostypes "github.com/GPTx-global/guru/types"
	evmtypes "github.com/GPTx-global/guru/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/tendermint/tendermint/libs/log"
	tmrpctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// BackendI implements the Cosmos and EVM backend.
type BackendI interface { //nolint: revive
	CosmosBackend
	EVMBackend
}

// CosmosBackend implements the functionality shared within cosmos namespaces
// as defined by Wallet Connect V2: https://docs.walletconnect.com/2.0/json-rpc/cosmos.
// Implemented by Backend.
type CosmosBackend interface { // TODO: define
	// GetAccounts()
	// SignDirect()
	// SignAmino()
}

// EVMBackend implements the functionality shared within ethereum namespaces
// as defined by EIP-1474: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1474.md
// Implemented by Backend.
type EVMBackend interface {
	// Node specific queries
	Accounts() ([]common.Address, error)
	Syncing() (interface{}, error)
	SetEtherbase(etherbase common.Address) bool
	SetGasPrice(gasPrice hexutil.Big) bool
	ImportRawKey(privkey, password string) (common.Address, error)
	ListAccounts() ([]common.Address, error)
	NewMnemonic(uid string, language keyring.Language, hdPath, bip39Passphrase string, algo keyring.SignatureAlgo) (*keyring.Record, error)
	UnprotectedAllowed() bool
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for send-transaction variants. The unit is ether.
	RPCMinGasPrice() int64

	// Sign Tx
	Sign(address common.Address, data hexutil.Bytes) (hexutil.Bytes, error)
	SendTransaction(args evmtypes.TransactionArgs) (common.Hash, error)
	SignTypedData(address common.Address, typedData apitypes.TypedData) (hexutil.Bytes, error)

	// Blocks Info
	BlockNumber() (hexutil.Uint64, error)
	GetBlockByNumber(blockNum rpctypes.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint
	TendermintBlockByNumber(blockNum rpctypes.BlockNumber) (*tmrpctypes.ResultBlock, error)
	TendermintBlockResultByNumber(height *int64) (*tmrpctypes.ResultBlockResults, error)
	TendermintBlockByHash(blockHash common.Hash) (*tmrpctypes.ResultBlock, error)
	BlockNumberFromTendermint(blockNrOrHash rpctypes.BlockNumberOrHash) (rpctypes.BlockNumber, error)
	BlockNumberFromTendermintByHash(blockHash common.Hash) (*big.Int, error)
	EthMsgsFromTendermintBlock(block *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) []*evmtypes.MsgEthereumTx
	BlockBloom(blockRes *tmrpctypes.ResultBlockResults) (ethtypes.Bloom, error)
	HeaderByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Header, error)
	HeaderByHash(blockHash common.Hash) (*ethtypes.Header, error)
	RPCBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults, fullTx bool) (map[string]interface{}, error)
	EthBlockByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Block, error)
	EthBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) (*ethtypes.Block, error)

	// Account Info
	GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error)
	GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetProof(address common.Address, storageKeys []string, blockNrOrHash rpctypes.BlockNumberOrHash) (*rpctypes.AccountResult, error)
	GetTransactionCount(address common.Address, blockNum rpctypes.BlockNumber) (*hexutil.Uint64, error)

	// Chain Info
	ChainID() (*hexutil.Big, error)
	ChainConfig() *params.ChainConfig
	GlobalMinGasPrice() (sdk.Dec, error)
	BaseFee(blockRes *tmrpctypes.ResultBlockResults) (*big.Int, error)
	CurrentHeader() *ethtypes.Header
	PendingTransactions() ([]*sdk.Tx, error)
	GetCoinbase() (sdk.AccAddress, error)
	FeeHistory(blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*rpctypes.FeeHistoryResult, error)
	SuggestGasTipCap(baseFee *big.Int) (*big.Int, error)

	// Tx Info
	GetTransactionByHash(txHash common.Hash) (*rpctypes.RPCTransaction, error)
	GetTxByEthHash(txHash common.Hash) (*evmostypes.TxResult, error)
	GetTxByTxIndex(height int64, txIndex uint) (*evmostypes.TxResult, error)
	GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)

	// Send Transaction
	Resend(args evmtypes.TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error)
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	SetTxDefaults(args evmtypes.TransactionArgs) (evmtypes.TransactionArgs, error)
	EstimateGas(args evmtypes.TransactionArgs, blockNrOptional *rpctypes.BlockNumber) (hexutil.Uint64, error)
	DoCall(args evmtypes.TransactionArgs, blockNr rpctypes.BlockNumber) (*evmtypes.MsgEthereumTxResponse, error)
	GasPrice() (*hexutil.Big, error)

	// Filter API
	GetLogs(hash common.Hash) ([][]*ethtypes.Log, error)
	GetLogsByHeight(height *int64) ([][]*ethtypes.Log, error)
	BloomStatus() (uint64, uint64)

	// Tracing
	TraceTransaction(hash common.Hash, config *evmtypes.TraceConfig) (interface{}, error)
	TraceBlock(height rpctypes.BlockNumber, config *evmtypes.TraceConfig, block *tmrpctypes.ResultBlock) ([]*evmtypes.TxTraceResult, error)
}

var _ BackendI = (*Backend)(nil)

var bAttributeKeyEthereumBloom = []byte(evmtypes.AttributeKeyEthereumBloom)

// Backend implements the BackendI interface
type Backend struct {
	ctx                 context.Context
	clientCtx           client.Context
	queryClient         *rpctypes.QueryClient // gRPC query client
	logger              log.Logger
	chainID             *big.Int
	cfg                 config.Config
	allowUnprotectedTxs bool
	indexer             evmostypes.EVMTxIndexer
}

// NewBackend creates a new Backend instance for cosmos and ethereum namespaces
func NewBackend(
	ctx *server.Context,
	logger log.Logger,
	clientCtx client.Context,
	allowUnprotectedTxs bool,
	indexer evmostypes.EVMTxIndexer,
) *Backend {
	chainID, err := evmostypes.ParseChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

	appConf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		panic(err)
	}

	return &Backend{
		ctx:                 context.Background(),
		clientCtx:           clientCtx,
		queryClient:         rpctypes.NewQueryClient(clientCtx),
		logger:              logger.With("module", "backend"),
		chainID:             chainID,
		cfg:                 appConf,
		allowUnprotectedTxs: allowUnprotectedTxs,
		indexer:             indexer,
	}
}
