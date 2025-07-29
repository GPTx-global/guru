package evm_test

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/GPTx-global/guru/utils"
	"github.com/GPTx-global/guru/x/evm/keeper"
	"github.com/status-im/keycard-go/hexutils"

	sdkmath "cosmossdk.io/math"
	"github.com/gogo/protobuf/proto"

	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/cosmos/cosmos-sdk/simapp"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	feemarkettypes "github.com/GPTx-global/guru/x/feemarket/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/crypto/ethsecp256k1"
	utiltx "github.com/GPTx-global/guru/testutil/tx"
	evmostypes "github.com/GPTx-global/guru/types"
	"github.com/GPTx-global/guru/x/evm"
	"github.com/GPTx-global/guru/x/evm/statedb"
	"github.com/GPTx-global/guru/x/evm/types"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/tendermint/tendermint/version"

	"github.com/GPTx-global/guru/testutil"
)

type EvmTestSuite struct {
	suite.Suite

	ctx     sdk.Context
	handler sdk.Handler
	app     *app.Guru
	chainID *big.Int

	signer    keyring.Signer
	ethSigner ethtypes.Signer
	from      common.Address
	to        sdk.AccAddress

	dynamicTxFee bool
}

func createInputWithMethidAndValue(method []byte, argValue []byte) []byte {
	argLength := big.NewInt(int64(len(argValue)))
	argOffset := big.NewInt(32)

	offsetBytes := make([]byte, 32)
	argOffset.FillBytes(offsetBytes)

	lengthBytes := make([]byte, 32)
	argLength.FillBytes(lengthBytes)

	valueBytes := make([]byte, 32)
	copy(valueBytes, argValue)

	input := append(method, offsetBytes...)
	input = append(input, lengthBytes...)
	input = append(input, valueBytes...)
	return input
}

// DoSetupTest setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *EvmTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())
	b32AccAddress := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
	suite.signer = utiltx.NewSigner(priv)
	suite.from = address
	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	consAddress := sdk.ConsAddress(priv.PubKey().Address())

	suite.app = app.EthSetup(checkTx, func(app *app.Guru, genesis simapp.GenesisState) simapp.GenesisState {
		if suite.dynamicTxFee {
			feemarketGenesis := feemarkettypes.DefaultGenesisState()
			feemarketGenesis.Params.EnableHeight = 1
			feemarketGenesis.Params.NoBaseFee = false
			genesis[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(feemarketGenesis)
		}
		return genesis
	})

	stakeDenom := stakingtypes.DefaultParams().BondDenom

	coins := sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdkmath.NewInt(100000000000000)), sdk.NewCoin(stakeDenom, sdkmath.NewInt(100000000000000)))
	genesisState := app.NewTestGenesisState(suite.app.AppCodec())
	b32address := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
	balances := []banktypes.Balance{
		{
			Address: b32AccAddress,
			Coins:   coins,
		},
		{
			Address: b32address,
			Coins:   coins,
		},
		{
			Address: suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
			Coins:   coins,
		},
	}
	var bankGenesis banktypes.GenesisState
	suite.app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	// Update balances and total supply
	bankGenesis.Balances = append(bankGenesis.Balances, balances...)
	bankGenesis.Supply = bankGenesis.Supply.Add(coins...).Add(coins...).Add(coins...)
	genesisState[banktypes.ModuleName] = suite.app.AppCodec().MustMarshalJSON(&bankGenesis)

	stateBytes, err := tmjson.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	// Initialize the chain
	req := abci.RequestInitChain{
		ChainId:         utils.TestnetChainID + "-1",
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: app.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	}
	suite.app.InitChain(req)

	suite.ctx = suite.app.BaseApp.NewContext(checkTx, tmproto.Header{
		Height:          1,
		ChainID:         req.ChainId,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)

	acc := &evmostypes.EthAccount{
		BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(address.Bytes()), nil, 0, 0),
		CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	}

	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	valAddr := sdk.ValAddress(address.Bytes())
	accAddr := sdk.AccAddress(address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr, priv.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)

	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	// err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	// require.NoError(t, err)
	suite.app.StakingKeeper.SetValidator(suite.ctx, validator)

	// Manually set indices for the first time
	// suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	// suite.app.StakingKeeper.SetValidatorByPowerIndex(suite.ctx, validator)

	// call the after-creation hook
	if err := suite.app.StakingKeeper.AfterValidatorCreated(suite.ctx, validator.GetOperator()); err != nil {
		panic(err)
	}
	_, err = suite.app.StakingKeeper.Delegate(suite.ctx, accAddr, sdkmath.NewInt(1), 1, validator, true)
	require.NoError(t, err)

	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
	suite.handler = evm.NewHandler(suite.app.EvmKeeper)

	// add distribution moderator and base
	suite.app.DistrKeeper.SetModeratorAddress(suite.ctx, "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h")
	suite.app.DistrKeeper.SetBaseAddress(suite.ctx, "guru1ks92ccc8sszwumjk2ue5v9rthlm2gp7ffx930h")
}

func (suite *EvmTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}

func (suite *EvmTestSuite) SignTx(tx *types.MsgEthereumTx) {
	tx.From = suite.from.String()
	err := tx.Sign(suite.ethSigner, suite.signer)
	suite.Require().NoError(err)
}

func (suite *EvmTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash().Bytes())))
}

func TestEvmTestSuite(t *testing.T) {
	suite.Run(t, new(EvmTestSuite))
}

func (suite *EvmTestSuite) TestHandleMsgEthereumTx() {
	var tx *types.MsgEthereumTx

	defaultEthTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    0,
		Amount:   big.NewInt(100),
		GasLimit: 0,
		GasPrice: big.NewInt(10000),
	}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"passed",
			func() {
				to := common.BytesToAddress(suite.to)
				ethTxParams := &types.EvmTxArgs{
					ChainID:  suite.chainID,
					Nonce:    0,
					To:       &to,
					Amount:   big.NewInt(10),
					GasLimit: 10_000_000,
					GasPrice: big.NewInt(10000),
				}
				tx = types.NewTx(ethTxParams)
				suite.SignTx(tx)
			},
			true,
		},
		{
			"insufficient balance",
			func() {
				tx = types.NewTx(defaultEthTxParams)
				suite.SignTx(tx)
			},
			false,
		},
		{
			"tx encoding failed",
			func() {
				tx = types.NewTx(defaultEthTxParams)
			},
			false,
		},
		{
			"invalid chain ID",
			func() {
				suite.ctx = suite.ctx.WithChainID("chainID")
			},
			false,
		},
		{
			"VerifySig failed",
			func() {
				tx = types.NewTx(defaultEthTxParams)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.handler(suite.ctx, tx)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *EvmTestSuite) TestHandlerLogs() {
	// Test contract:

	// pragma solidity ^0.5.1;

	// contract Test {
	//     event Hello(uint256 indexed world);

	//     constructor() public {
	//         emit Hello(17);
	//     }
	// }

	// {
	// 	"linkReferences": {},
	// 	"object": "6080604052348015600f57600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a2603580604b6000396000f3fe6080604052600080fdfea165627a7a723058206cab665f0f557620554bb45adf266708d2bd349b8a4314bdff205ee8440e3c240029",
	// 	"opcodes": "PUSH1 0x80 PUSH1 0x40 MSTORE CALLVALUE DUP1 ISZERO PUSH1 0xF JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST POP PUSH1 0x11 PUSH32 0x775A94827B8FD9B519D36CD827093C664F93347070A554F65E4A6F56CD738898 PUSH1 0x40 MLOAD PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 LOG2 PUSH1 0x35 DUP1 PUSH1 0x4B PUSH1 0x0 CODECOPY PUSH1 0x0 RETURN INVALID PUSH1 0x80 PUSH1 0x40 MSTORE PUSH1 0x0 DUP1 REVERT INVALID LOG1 PUSH6 0x627A7A723058 KECCAK256 PUSH13 0xAB665F0F557620554BB45ADF26 PUSH8 0x8D2BD349B8A4314 0xbd SELFDESTRUCT KECCAK256 0x5e 0xe8 DIFFICULTY 0xe EXTCODECOPY 0x24 STOP 0x29 ",
	// 	"sourceMap": "25:119:0:-;;;90:52;8:9:-1;5:2;;;30:1;27;20:12;5:2;90:52:0;132:2;126:9;;;;;;;;;;25:119;;;;;;"
	// }

	gasLimit := uint64(100000)
	gasPrice := big.NewInt(1000000)

	bytecode := common.FromHex("0x6080604052348015600f57600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a2603580604b6000396000f3fe6080604052600080fdfea165627a7a723058206cab665f0f557620554bb45adf266708d2bd349b8a4314bdff205ee8440e3c240029")

	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	var txResponse types.MsgEthereumTxResponse

	err = proto.Unmarshal(result.Data, &txResponse)
	suite.Require().NoError(err, "failed to decode result data")

	suite.Require().Equal(len(txResponse.Logs), 1)
	suite.Require().Equal(len(txResponse.Logs[0].Topics), 2)
}

func (suite *EvmTestSuite) TestDeployAndCallContract() {
	// Test contract:
	// http://remix.ethereum.org/#optimize=false&evmVersion=istanbul&version=soljson-v0.5.15+commit.6a57276f.js
	// 2_Owner.sol
	//
	// pragma solidity >=0.4.22 <0.7.0;
	//
	///**
	// * @title Owner
	// * @dev Set & change owner
	// */
	// contract Owner {
	//
	//	address private owner;
	//
	//	// event for EVM logging
	//	event OwnerSet(address indexed oldOwner, address indexed newOwner);
	//
	//	// modifier to check if caller is owner
	//	modifier isOwner() {
	//	// If the first argument of 'require' evaluates to 'false', execution terminates and all
	//	// changes to the state and to Ether balances are reverted.
	//	// This used to consume all gas in old EVM versions, but not anymore.
	//	// It is often a good idea to use 'require' to check if functions are called correctly.
	//	// As a second argument, you can also provide an explanation about what went wrong.
	//	require(msg.sender == owner, "Caller is not owner");
	//	_;
	//}
	//
	//	/**
	//	 * @dev Set contract deployer as owner
	//	 */
	//	constructor() public {
	//	owner = msg.sender; // 'msg.sender' is sender of current call, contract deployer for a constructor
	//	emit OwnerSet(address(0), owner);
	//}
	//
	//	/**
	//	 * @dev Change owner
	//	 * @param newOwner address of new owner
	//	 */
	//	function changeOwner(address newOwner) public isOwner {
	//	emit OwnerSet(owner, newOwner);
	//	owner = newOwner;
	//}
	//
	//	/**
	//	 * @dev Return owner address
	//	 * @return address of owner
	//	 */
	//	function getOwner() external view returns (address) {
	//	return owner;
	//}
	//}

	// Deploy contract - Owner.sol
	gasLimit := uint64(100000000)
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a36102c4806100dc6000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c010000000000000000000000000000000000000000000000000000000090048063893d20e814610058578063a6f9dae1146100a2575b600080fd5b6100606100e6565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100e4600480360360208110156100b857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061010f565b005b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101d1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f43616c6c6572206973206e6f74206f776e65720000000000000000000000000081525060200191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fea265627a7a72315820f397f2733a89198bc7fed0764083694c5b828791f39ebcbc9e414bccef14b48064736f6c63430005100032")
	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	var res types.MsgEthereumTxResponse

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// store - changeOwner
	gasLimit = uint64(100000000000)
	gasPrice = big.NewInt(100)
	receiver := crypto.CreateAddress(suite.from, 1)

	// keccak256("changeOwner(address)") => a6f9dae1
	storeAddr := "0xa6f9dae10000000000000000000000006a82e4a67715c8412a9114fbd2cbaefbc8181424"
	bytecode = common.FromHex(storeAddr)

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// query - getOwner
	bytecode = common.FromHex("0x893d20e8")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	getAddr := strings.ToLower(hexutils.BytesToHex(res.Ret))
	fmt.Println("getAddr", getAddr)

	// FIXME: correct owner?
	// getAddr := strings.ToLower(hexutils.BytesToHex(res.Ret))
	// suite.Require().Equal(true, strings.HasSuffix(storeAddr, getAddr), "Fail to query the address")
}

func (suite *EvmTestSuite) TestSendTransaction() {
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(0x55ae82600)

	// send simple value transfer with gasLimit=21000
	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		To:       &common.Address{0x1},
		Amount:   big.NewInt(1),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)
}

func (suite *EvmTestSuite) TestOutOfGasWhenDeployContract() {
	// Test contract:
	// http://remix.ethereum.org/#optimize=false&evmVersion=istanbul&version=soljson-v0.5.15+commit.6a57276f.js
	// 2_Owner.sol
	//
	// pragma solidity >=0.4.22 <0.7.0;
	//
	///**
	// * @title Owner
	// * @dev Set & change owner
	// */
	// contract Owner {
	//
	//	address private owner;
	//
	//	// event for EVM logging
	//	event OwnerSet(address indexed oldOwner, address indexed newOwner);
	//
	//	// modifier to check if caller is owner
	//	modifier isOwner() {
	//	// If the first argument of 'require' evaluates to 'false', execution terminates and all
	//	// changes to the state and to Ether balances are reverted.
	//	// This used to consume all gas in old EVM versions, but not anymore.
	//	// It is often a good idea to use 'require' to check if functions are called correctly.
	//	// As a second argument, you can also provide an explanation about what went wrong.
	//	require(msg.sender == owner, "Caller is not owner");
	//	_;
	//}
	//
	//	/**
	//	 * @dev Set contract deployer as owner
	//	 */
	//	constructor() public {
	//	owner = msg.sender; // 'msg.sender' is sender of current call, contract deployer for a constructor
	//	emit OwnerSet(address(0), owner);
	//}
	//
	//	/**
	//	 * @dev Change owner
	//	 * @param newOwner address of new owner
	//	 */
	//	function changeOwner(address newOwner) public isOwner {
	//	emit OwnerSet(owner, newOwner);
	//	owner = newOwner;
	//}
	//
	//	/**
	//	 * @dev Return owner address
	//	 * @return address of owner
	//	 */
	//	function getOwner() external view returns (address) {
	//	return owner;
	//}
	//}

	// Deploy contract - Owner.sol
	gasLimit := uint64(1)
	suite.ctx = suite.ctx.WithGasMeter(sdk.NewGasMeter(gasLimit))
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a36102c4806100dc6000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c010000000000000000000000000000000000000000000000000000000090048063893d20e814610058578063a6f9dae1146100a2575b600080fd5b6100606100e6565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100e4600480360360208110156100b857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061010f565b005b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101d1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f43616c6c6572206973206e6f74206f776e65720000000000000000000000000081525060200191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fea265627a7a72315820f397f2733a89198bc7fed0764083694c5b828791f39ebcbc9e414bccef14b48064736f6c63430005100032")
	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	defer func() {
		//nolint:revive // allow empty code block that just contains TODO in test code
		if r := recover(); r != nil {
			// TODO: snapshotting logic
		} else {
			suite.Require().Fail("panic did not happen")
		}
	}()

	_, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err)

	suite.Require().Fail("panic did not happen")
}

func (suite *EvmTestSuite) TestErrorWhenDeployContract() {
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0xa6f9dae10000000000000000000000006a82e4a67715c8412a9114fbd2cbaefbc8181424")

	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, _ := suite.handler(suite.ctx, tx)
	var res types.MsgEthereumTxResponse

	_ = proto.Unmarshal(result.Data, &res)

	suite.Require().Equal("invalid opcode: opcode 0xa6 not defined", res.VmError, "correct evm error")

	// TODO: snapshot checking
}

func (suite *EvmTestSuite) deployERC20Contract() common.Address {
	k := suite.app.EvmKeeper
	nonce := k.GetNonce(suite.ctx, suite.from)
	ctorArgs, err := types.ERC20Contract.ABI.Pack("", suite.from, big.NewInt(10000000000))
	suite.Require().NoError(err)
	msg := core.NewMessage(
		suite.from,
		nil,
		nonce,
		big.NewInt(0),
		2000000,
		big.NewInt(1),
		nil,
		nil,
		append(types.ERC20Contract.Bin, ctorArgs...),
		nil,
		true,
	)
	rsp, err := k.ApplyMessage(suite.ctx, msg, nil, true)
	suite.Require().NoError(err)
	suite.Require().False(rsp.Failed())
	return crypto.CreateAddress(suite.from, nonce)
}

// TestERC20TransferReverted checks:
// - when transaction reverted, gas refund works.
// - when transaction reverted, nonce is still increased.
func (suite *EvmTestSuite) TestERC20TransferReverted() {
	intrinsicGas := uint64(21572)
	// test different hooks scenarios
	testCases := []struct {
		msg      string
		gasLimit uint64
		hooks    types.EvmHooks
		expErr   string
	}{
		{
			"no hooks",
			intrinsicGas, // enough for intrinsicGas, but not enough for execution
			nil,
			"out of gas",
		},
		{
			"success hooks",
			intrinsicGas, // enough for intrinsicGas, but not enough for execution
			&DummyHook{},
			"out of gas",
		},
		{
			"failure hooks",
			1000000, // enough gas limit, but hooks fails.
			&FailureHook{},
			"failed to execute post processing",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			suite.SetupTest()
			k := suite.app.EvmKeeper.CleanHooks()
			k.SetHooks(tc.hooks)

			// add some fund to pay gas fee
			err := k.SetBalance(suite.ctx, suite.from, big.NewInt(1000000000000000))
			suite.Require().NoError(err)

			contract := suite.deployERC20Contract()

			data, err := types.ERC20Contract.ABI.Pack("transfer", suite.from, big.NewInt(10))
			suite.Require().NoError(err)

			gasPrice := big.NewInt(1000000000) // must be bigger than or equal to baseFee
			nonce := k.GetNonce(suite.ctx, suite.from)
			ethTxParams := &types.EvmTxArgs{
				ChainID:  suite.chainID,
				Nonce:    nonce,
				To:       &contract,
				Amount:   big.NewInt(0),
				GasPrice: gasPrice,
				GasLimit: tc.gasLimit,
				Input:    data,
			}
			tx := types.NewTx(ethTxParams)
			suite.SignTx(tx)

			before := k.GetBalance(suite.ctx, suite.from)

			evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := evmParams.GetChainConfig().EthereumConfig(nil)
			baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)

			txData, err := types.UnpackTxData(tx.Data)
			suite.Require().NoError(err)
			fees, err := keeper.VerifyFee(txData, types.DefaultEVMDenom, baseFee, true, true, true, suite.ctx.IsCheckTx())
			suite.Require().NoError(err)
			err = k.DeductTxCostsFromUserBalance(suite.ctx, fees, common.HexToAddress(tx.From))
			suite.Require().NoError(err)

			res, err := k.EthereumTx(sdk.WrapSDKContext(suite.ctx), tx)
			suite.Require().NoError(err)

			suite.Require().True(res.Failed())
			suite.Require().Equal(tc.expErr, res.VmError)
			suite.Require().Empty(res.Logs)

			after := k.GetBalance(suite.ctx, suite.from)

			if tc.expErr == "out of gas" {
				suite.Require().Equal(tc.gasLimit, res.GasUsed)
			} else {
				suite.Require().Greater(tc.gasLimit, res.GasUsed)
			}

			// check gas refund works: only deducted fee for gas used, rather than gas limit.
			suite.Require().Equal(new(big.Int).Mul(gasPrice, big.NewInt(int64(res.GasUsed))), new(big.Int).Sub(before, after))

			// nonce should not be increased.
			nonce2 := k.GetNonce(suite.ctx, suite.from)
			suite.Require().Equal(nonce, nonce2)
		})
	}
}

func (suite *EvmTestSuite) TestContractDeploymentRevert() {
	intrinsicGas := uint64(234180) //134180
	testCases := []struct {
		msg      string
		gasLimit uint64
		hooks    types.EvmHooks
	}{
		{
			"no hooks",
			intrinsicGas,
			nil,
		},
		{
			"success hooks",
			intrinsicGas,
			&DummyHook{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			suite.SetupTest()
			k := suite.app.EvmKeeper.CleanHooks()

			// test with different hooks scenarios
			k.SetHooks(tc.hooks)

			nonce := k.GetNonce(suite.ctx, suite.from)
			ctorArgs, err := types.ERC20Contract.ABI.Pack("", suite.from, big.NewInt(0))
			suite.Require().NoError(err)

			ethTxParams := &types.EvmTxArgs{
				Nonce:    nonce,
				GasLimit: tc.gasLimit,
				Input:    append(types.ERC20Contract.Bin, ctorArgs...),
			}
			tx := types.NewTx(ethTxParams)
			suite.SignTx(tx)

			// simulate nonce increment in ante handler
			db := suite.StateDB()
			db.SetNonce(suite.from, nonce+1)
			suite.Require().NoError(db.Commit())

			rsp, err := k.EthereumTx(sdk.WrapSDKContext(suite.ctx), tx)
			suite.Require().NoError(err)
			suite.Require().True(rsp.Failed())

			// nonce don't change
			nonce2 := k.GetNonce(suite.ctx, suite.from)
			suite.Require().Equal(nonce+1, nonce2)
		})
	}
}

// DummyHook implements EvmHooks interface
type DummyHook struct{}

func (dh *DummyHook) PostTxProcessing(_ sdk.Context, _ core.Message, _ *ethtypes.Receipt) error {
	return nil
}

// FailureHook implements EvmHooks interface
type FailureHook struct{}

func (dh *FailureHook) PostTxProcessing(_ sdk.Context, _ core.Message, _ *ethtypes.Receipt) error {
	return errors.New("mock error")
}

func (suite *EvmTestSuite) TestEIP5656_MCOPY() {
	// Test contract:
	//
	// // SPDX-License-Identifier: MIT
	// pragma solidity ^0.8.24;
	//
	// contract MCopyExample {
	//     bytes public copyData;
	//
	//     function copyMemory(bytes memory source) public returns (bytes memory) {
	//         bytes memory destination = new bytes(source.length);
	//
	//         assembly {
	//             mcopy(
	//                 add(destination, 0x20),
	//                 add(source, 0x20),
	//                 mload(source)
	//             )
	//         }
	//
	//         copyData = destination;
	//
	//         return destination;
	//     }
	//
	//     function getData() public view returns (bytes memory) {
	//         return copyData;
	//     }
	// }
	//
	// bytecode: 6080604052348015600e575f5ffd5b5061075e8061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c80632f703e08146100435780633bc5de3014610061578063c83aa2f31461007f575b5f5ffd5b61004b6100af565b60405161005891906102af565b60405180910390f35b61006961013a565b60405161007691906102af565b60405180910390f35b6100996004803603810190610094919061040c565b6101c9565b6040516100a691906102af565b60405180910390f35b5f80546100bb90610480565b80601f01602080910402602001604051908101604052809291908181526020018280546100e790610480565b80156101325780601f1061010957610100808354040283529160200191610132565b820191905f5260205f20905b81548152906001019060200180831161011557829003601f168201915b505050505081565b60605f805461014890610480565b80601f016020809104026020016040519081016040528092919081815260200182805461017490610480565b80156101bf5780601f10610196576101008083540402835291602001916101bf565b820191905f5260205f20905b8154815290600101906020018083116101a257829003601f168201915b5050505050905090565b60605f825167ffffffffffffffff8111156101e7576101e66102e8565b5b6040519080825280601f01601f1916602001820160405280156102195781602001600182028036833780820191505090505b509050825160208401602083015e805f90816102359190610659565b5080915050919050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6102818261023f565b61028b8185610249565b935061029b818560208601610259565b6102a481610267565b840191505092915050565b5f6020820190508181035f8301526102c78184610277565b905092915050565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61031e82610267565b810181811067ffffffffffffffff8211171561033d5761033c6102e8565b5b80604052505050565b5f61034f6102cf565b905061035b8282610315565b919050565b5f67ffffffffffffffff82111561037a576103796102e8565b5b61038382610267565b9050602081019050919050565b828183375f83830152505050565b5f6103b06103ab84610360565b610346565b9050828152602081018484840111156103cc576103cb6102e4565b5b6103d7848285610390565b509392505050565b5f82601f8301126103f3576103f26102e0565b5b813561040384826020860161039e565b91505092915050565b5f60208284031215610421576104206102d8565b5b5f82013567ffffffffffffffff81111561043e5761043d6102dc565b5b61044a848285016103df565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061049757607f821691505b6020821081036104aa576104a9610453565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f6008830261050c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826104d1565b61051686836104d1565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f61055a6105556105508461052e565b610537565b61052e565b9050919050565b5f819050919050565b61057383610540565b61058761057f82610561565b8484546104dd565b825550505050565b5f5f905090565b61059e61058f565b6105a981848461056a565b505050565b5b818110156105cc576105c15f82610596565b6001810190506105af565b5050565b601f821115610611576105e2816104b0565b6105eb846104c2565b810160208510156105fa578190505b61060e610606856104c2565b8301826105ae565b50505b505050565b5f82821c905092915050565b5f6106315f1984600802610616565b1980831691505092915050565b5f6106498383610622565b9150826002028217905092915050565b6106628261023f565b67ffffffffffffffff81111561067b5761067a6102e8565b5b6106858254610480565b6106908282856105d0565b5f60209050601f8311600181146106c1575f84156106af578287015190505b6106b9858261063e565b865550610720565b601f1984166106cf866104b0565b5f5b828110156106f6578489015182556001820191506020850194506020810190506106d1565b86831015610713578489015161070f601f891682610622565b8355505b6001600288020188555050505b50505050505056fea2646970667358221220ca9668f26243808a963af74de512261559d1275573a383f803ae55b7e30bd20564736f6c634300081d0033
	// "2f703e08": "copyData()",
	// "c83aa2f3": "copyMemory(bytes)",
	// "3bc5de30": "getData()"

	// Deploy contract - MCopyExample
	gasLimit := uint64(100000000)
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("6080604052348015600e575f80fd5b5061075b8061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061003f575f3560e01c80632f703e08146100435780633bc5de3014610061578063c83aa2f31461007f575b5f80fd5b61004b6100af565b60405161005891906102af565b60405180910390f35b61006961013a565b60405161007691906102af565b60405180910390f35b6100996004803603810190610094919061040c565b6101c9565b6040516100a691906102af565b60405180910390f35b5f80546100bb90610480565b80601f01602080910402602001604051908101604052809291908181526020018280546100e790610480565b80156101325780601f1061010957610100808354040283529160200191610132565b820191905f5260205f20905b81548152906001019060200180831161011557829003601f168201915b505050505081565b60605f805461014890610480565b80601f016020809104026020016040519081016040528092919081815260200182805461017490610480565b80156101bf5780601f10610196576101008083540402835291602001916101bf565b820191905f5260205f20905b8154815290600101906020018083116101a257829003601f168201915b5050505050905090565b60605f825167ffffffffffffffff8111156101e7576101e66102e8565b5b6040519080825280601f01601f1916602001820160405280156102195781602001600182028036833780820191505090505b509050825160208401602083015e805f90816102359190610656565b5080915050919050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6102818261023f565b61028b8185610249565b935061029b818560208601610259565b6102a481610267565b840191505092915050565b5f6020820190508181035f8301526102c78184610277565b905092915050565b5f604051905090565b5f80fd5b5f80fd5b5f80fd5b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61031e82610267565b810181811067ffffffffffffffff8211171561033d5761033c6102e8565b5b80604052505050565b5f61034f6102cf565b905061035b8282610315565b919050565b5f67ffffffffffffffff82111561037a576103796102e8565b5b61038382610267565b9050602081019050919050565b828183375f83830152505050565b5f6103b06103ab84610360565b610346565b9050828152602081018484840111156103cc576103cb6102e4565b5b6103d7848285610390565b509392505050565b5f82601f8301126103f3576103f26102e0565b5b813561040384826020860161039e565b91505092915050565b5f60208284031215610421576104206102d8565b5b5f82013567ffffffffffffffff81111561043e5761043d6102dc565b5b61044a848285016103df565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061049757607f821691505b6020821081036104aa576104a9610453565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f6008830261050c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826104d1565b61051686836104d1565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f61055a6105556105508461052e565b610537565b61052e565b9050919050565b5f819050919050565b61057383610540565b61058761057f82610561565b8484546104dd565b825550505050565b5f90565b61059b61058f565b6105a681848461056a565b505050565b5b818110156105c9576105be5f82610593565b6001810190506105ac565b5050565b601f82111561060e576105df816104b0565b6105e8846104c2565b810160208510156105f7578190505b61060b610603856104c2565b8301826105ab565b50505b505050565b5f82821c905092915050565b5f61062e5f1984600802610613565b1980831691505092915050565b5f610646838361061f565b9150826002028217905092915050565b61065f8261023f565b67ffffffffffffffff811115610678576106776102e8565b5b6106828254610480565b61068d8282856105cd565b5f60209050601f8311600181146106be575f84156106ac578287015190505b6106b6858261063b565b86555061071d565b601f1984166106cc866104b0565b5f5b828110156106f3578489015182556001820191506020850194506020810190506106ce565b86831015610710578489015161070c601f89168261061f565b8355505b6001600288020188555050505b50505050505056fea264697066735822122030d42ff52cb7d83dd87a0c04c38e75d22c09b9581f0d624c4d38fed76ff596ba64736f6c634300081a0033")
	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	var res types.MsgEthereumTxResponse

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// Call copyMemory
	gasLimit = uint64(100000000000)
	gasPrice = big.NewInt(100)
	receiver := crypto.CreateAddress(suite.from, 1)

	// ABI Encoding: Method(4bytes) + Arg offset(32bytes) + Arg offset(32bytes) + Arg value(include padding 32bytes)
	// keccak-256("copyMemory(bytes)") => c83aa2f325bdbf4402c849d4b1952e56ad2f8a385d6710cf232a2828779d645e
	bytecode = common.FromHex("0xc83aa2f3")
	argValue := common.FromHex("0xff")
	input := createInputWithMethidAndValue(bytecode, argValue)

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    input,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// Call getData
	bytecode = common.FromHex("0x3bc5de30")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    3,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	data := strings.ToLower(hexutils.BytesToHex(res.Ret))
	//32byetes + 32bytes + 32bytes
	//0x20 = 32
	//0x01 = 1
	suite.Require().Equal(data, "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001ff00000000000000000000000000000000000000000000000000000000000000", "failed to call getData")
	// fmt.Println("Data:", data)
}

func (suite *EvmTestSuite) TestEIP1153_TransientStorage() {
	// Test contract:
	// // SPDX-License-Identifier: MIT
	// pragma solidity ^0.8.28;
	//
	// contract TransientStorageTest {
	//     uint transient tStore;
	//     uint public nStore; // Normal storage variable
	//
	//     // Private function to store value in transient storage
	//     function store(uint _v) private {
	//         tStore = _v;
	//     }
	//
	//     // Public function to test transient storage within a single transaction
	//     function testTransientStorage(uint _in) public returns (uint256) {
	//         store(_in); // Store input in transient storage
	//         nStore = tStore; // Copy transient storage to normal storage
	//         return tStore; // Return value from transient storage
	//     }
	//
	//     // Public function to test transient storage in a subsequent transaction
	//     function testTransientStorage2() public returns (uint256) {
	//         nStore = tStore; // Copy transient storage (should be 0) to normal storage
	//         return tStore; // Return value from transient storage (should be 0)
	//     }
	// }
	// Bytecode: 6080604052348015600e575f5ffd5b506101a68061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c806323be0d1014610043578063363b976e14610061578063f63f3a9914610091575b5f5ffd5b61004b6100af565b60405161005891906100fe565b60405180910390f35b61007b60048036038101906100769190610145565b6100be565b60405161008891906100fe565b60405180910390f35b6100996100d8565b6040516100a691906100fe565b60405180910390f35b5f5f5c5f819055505f5c905090565b5f6100c8826100dd565b5f5c5f819055505f5c9050919050565b5f5481565b805f81905d5050565b5f819050919050565b6100f8816100e6565b82525050565b5f6020820190506101115f8301846100ef565b92915050565b5f5ffd5b610124816100e6565b811461012e575f5ffd5b50565b5f8135905061013f8161011b565b92915050565b5f6020828403121561015a57610159610117565b5b5f61016784828501610131565b9150509291505056fea2646970667358221220f66c263c5fa7352912625828ca030bd498d350dd60249338bc8a16078e2cb6ce64736f6c634300081c0033
	// testTransientStorage(uint256) selector: 0x363b976e
	// testTransientStorage2() selector: 0x23be0d10
	// nStore() selector: 0xf63f3a99

	// Deploy contract
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(10000)
	// Updated Bytecode for the new contract
	bytecode := common.FromHex("6080604052348015600e575f5ffd5b506101a68061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c806323be0d1014610043578063363b976e14610061578063f63f3a9914610091575b5f5ffd5b61004b6100af565b60405161005891906100fe565b60405180910390f35b61007b60048036038101906100769190610145565b6100be565b60405161008891906100fe565b60405180910390f35b6100996100d8565b6040516100a691906100fe565b60405180910390f35b5f5f5c5f819055505f5c905090565b5f6100c8826100dd565b5f5c5f819055505f5c9050919050565b5f5481565b805f81905d5050565b5f819050919050565b6100f8816100e6565b82525050565b5f6020820190506101115f8301846100ef565b92915050565b5f5ffd5b610124816100e6565b811461012e575f5ffd5b50565b5f8135905061013f8161011b565b92915050565b5f6020828403121561015a57610159610117565b5b5f61016784828501610131565b9150509291505056fea2646970667358221220f66c263c5fa7352912625828ca030bd498d350dd60249338bc8a16078e2cb6ce64736f6c634300081c0033")

	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1, // Assuming nonce starts at 1 after setup
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle contract deployment tx")

	var res types.MsgEthereumTxResponse
	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode deployment result data")
	suite.Require().Equal(res.VmError, "", "contract deployment failed")
	suite.Require().NotEmpty(res.Ret, "contract deployment returned empty address")
	// contractAddr := common.BytesToAddress(res.Ret) // Get the deployed contract address
	contractAddr := crypto.CreateAddress(suite.from, 1)

	// fmt.Println("Contract Address:", contractAddr.Hex())
	// fmt.Println("receiver Address:", receiver.Hex())

	// Call testTransientStorage(42)
	valueToStore := big.NewInt(42)
	testTransientStorageSelector := common.FromHex("0x363b976e") // Updated selector

	valueBytes := make([]byte, 32)
	valueToStore.FillBytes(valueBytes)

	inputData := append(testTransientStorageSelector, valueBytes...)

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    2, // Increment nonce for the next transaction
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    inputData,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle testTransientStorage tx")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode testTransientStorage result data")
	// Expect no VM error now that Shanghai should be active
	suite.Require().Equal("", res.VmError, "testTransientStorage call failed")
	suite.Require().NotEmpty(res.Ret, "testTransientStorage returned empty data")

	returnedValue := new(big.Int).SetBytes(res.Ret)
	suite.Require().Equal(valueToStore, returnedValue, "testTransientStorage returned incorrect value")

	// Call nStore() to check normal storage
	nStoreSelector := common.FromHex("0xf63f3a99") // Updated selector
	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    3, // Increment nonce
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    nStoreSelector,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle nStore read tx")
	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode nStore read result data")
	suite.Require().Equal("", res.VmError, "nStore read call failed")
	suite.Require().NotEmpty(res.Ret, "nStore read returned empty data")

	nStoreValue := new(big.Int).SetBytes(res.Ret)
	suite.Require().Equal(valueToStore, nStoreValue, "nStore has incorrect value after testTransientStorage")

	// Call testTransientStorage2() in a NEW transaction
	testTransientStorage2Selector := common.FromHex("0x23be0d10") // Updated selector

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    4, // Increment nonce again
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    testTransientStorage2Selector,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle testTransientStorage2 tx")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode testTransientStorage2 result data")
	suite.Require().Equal("", res.VmError, "testTransientStorage2 call failed")
	suite.Require().NotEmpty(res.Ret, "testTransientStorage2 returned empty data")

	returnedValueAfter := new(big.Int).SetBytes(res.Ret)
	// tStore should be 0 in a new transaction
	suite.Require().Equal(uint64(0), returnedValueAfter.Uint64(), "testTransientStorage2 returned non-zero value from tStore")

	// Call nStore() again to check normal storage after testTransientStorage2
	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    5, // Increment nonce
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    nStoreSelector, // Reuse selector
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle second nStore read tx")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode second nStore read result data")
	suite.Require().Equal("", res.VmError, "second nStore read call failed")
	suite.Require().NotEmpty(res.Ret, "second nStore read returned empty data")

	nStoreValueAfter := new(big.Int).SetBytes(res.Ret)
	// nStore should have been updated to 0 by testTransientStorage2
	suite.Require().Equal(uint64(0), nStoreValueAfter.Uint64(), "nStore has incorrect value after testTransientStorage2")
}

func (suite *EvmTestSuite) TestEIP6780_Selfdestruct() {
	// Test contract:
	// // SPDX-License-Identifier: MIT
	// pragma solidity ^0.8.20;
	//
	// /**
	//  * @title Victim Contract
	//  * @notice This contract is designed to be destroyed using SELFDESTRUCT.
	//  * It contains a function to trigger selfdestruct and can receive Ether.
	//  */
	// contract Victim {
	//     event Destroyed(address indexed target, uint amount);
	//
	//     /**
	//      * @notice Constructor is marked payable to allow receiving Ether during creation.
	//      */
	//     constructor() payable {
	//         // 생성 시 특별히 할 작업이 없더라도 payable로 선언해야
	//         // new Victim{value: ...}() 구문으로 ETH를 받을 수 있습니다.
	//     }
	//
	//     // Allow receiving Ether so it has a balance to transfer on selfdestruct
	//     receive() external payable {}
	//
	//     /**
	//      * @notice Triggers the selfdestruct opcode, attempting to send
	//      * the contract's balance to the specified target address.
	//      * @param target The address to receive the contract's Ether balance.
	//      */
	//     function destroy(address payable target) public {
	//         uint balance = address(this).balance;
	//         emit Destroyed(target, balance);
	//         // The core opcode being tested
	//         selfdestruct(target);
	//     }
	//
	//     /**
	//      * @notice A simple function to check if the contract's code is still executable.
	//      * If the contract is destroyed, calling this function (externally) should fail
	//      * or behave as if calling an EOA with no code.
	//      * @return bool Always returns true if the code is executing.
	//      */
	//     function isAlive() public pure returns (bool) {
	//         return true;
	//     }
	// }
	//
	// /**
	//  * @title EIP-6780 Tester Contract
	//  * @notice This contract facilitates testing the behavior of SELFDESTRUCT
	//  * under EIP-6780 rules by running two scenarios.
	//  */
	// contract Tester {
	//     address public victimAddressForScenarioA; // Scenario A에서 사용할 Victim 컨트랙트 주소 저장
	//     address public victimAddressForScenarioB; // Scenario B에서 사용할 Victim 컨트랙트 주소 저장
	//
	//     // Tester 컨트랙트가 ETH를 받을 수 있도록 함 (Victim 자금 조달 및 selfdestruct로부터 자금 수신)
	//     receive() external payable {}
	//
	//     // --- 시나리오 A: 같은 트랜잭션에서 생성 및 파괴 ---
	//     /**
	//      * @notice Scenario A: Creates a Victim contract and immediately calls its
	//      * 'destroy' function within the SAME transaction.
	//      * Sends any received Ether to the Victim upon creation.
	//      * The Victim attempts to selfdestruct, sending its balance back to this Tester contract.
	//      * @dev According to EIP-6780, the Victim contract SHOULD be destroyed.
	//      */
	//     function testCreateAndDestroySameTx() public payable {
	//         // 1. Victim 컨트랙트 생성 (이 함수로 전송된 ETH 전달)
	//         Victim victim = new Victim{value: msg.value}();
	//         address victimAddress = address(victim);
	//
	//         victimAddressForScenarioA = victimAddress;
	//
	//         // 2. 즉시 Victim의 destroy 함수 호출 (자금을 이 Tester 컨트랙트로 다시 보냄)
	//         // selfdestruct의 대상 주소는 이 Tester 컨트랙트
	//         victim.destroy(payable(address(this)));
	//
	//         // 참고: 동일 트랜잭션 내에서 selfdestruct 직후 코드 크기 확인은 EVM 실행 순서에 따라
	//         // 예상과 다를 수 있습니다. 트랜잭션 완료 *후* 외부에서 확인하는 것이 더 확실합니다.
	//         // 이 시나리오에서는 victimAddress의 코드가 0이 될 것으로 예상합니다.
	//     }
	//
	//     // --- 시나리오 B: 다른 트랜잭션에서 생성 후 파괴 ---
	//
	//     /**
	//      * @notice Scenario B - Step 1: Creates a Victim contract and stores its address.
	//      * Sends any received Ether to the Victim upon creation.
	//      * This transaction ONLY creates the Victim.
	//      */
	//     function createVictimForScenarioB() public payable {
	//         Victim victim = new Victim{value: msg.value}();
	//         victimAddressForScenarioB = address(victim);
	//     }
	//
	//     /**
	//      * @notice Scenario B - Step 2: Calls the 'destroy' function on the Victim
	//      * contract created in a PREVIOUS transaction (via createVictimForScenarioB).
	//      * This happens in a SEPARATE transaction from the creation.
	//      * @dev According to EIP-6780, the Victim contract SHOULD NOT be destroyed,
	//      * but its Ether balance SHOULD be transferred to this Tester contract.
	//      */
	//     function destroyVictimFromScenarioB() public {
	//         require(victimAddressForScenarioB != address(0), "Victim for Scenario B not created yet");
	//
	//         Victim victimInstance = Victim(payable(victimAddressForScenarioB));
	//
	//         // 대상 Victim의 destroy 함수 호출
	//         // selfdestruct의 대상 주소는 이 Tester 컨트랙트
	//         victimInstance.destroy(payable(address(this)));
	//
	//         // 이 시나리오에서는 victimAddressForScenarioB의 코드가 0이 아니어야 합니다. (파괴되지 않음)
	//         // 하지만 ETH 잔액은 이 Tester 컨트랙트로 전송되어야 합니다.
	//     }
	//
	//     // --- 검증 헬퍼 함수 (트랜잭션 실행 후 외부에서 호출하여 확인) ---
	//
	//     /**
	//      * @notice Checks the code size of a given address.
	//      * @param _addr The address to check.
	//      * @return uint The size of the code at the address (0 if destroyed or EOA).
	//      */
	//     function getCodeSize(address _addr) public view returns (uint) {
	//         return _addr.code.length;
	//     }
	//
	//     /**
	//      * @notice Checks the Ether balance of a given address.
	//      * @param _addr The address to check.
	//      * @return uint The Ether balance in Wei.
	//      */
	//     function getBalance(address _addr) public view returns (uint) {
	//         return _addr.balance;
	//     }
	//
	//     /**
	//      * @notice Checks if the Victim from Scenario B still has code (is alive).
	//      * Call this AFTER running destroyVictimFromScenarioB().
	//      * @return bool True if the Victim contract still has code, false otherwise.
	//      */
	//     function isVictimBAlive() public view returns (bool) {
	//          if (victimAddressForScenarioB == address(0)) {
	//              return false; // 아직 생성되지 않음
	//          }
	//          return getCodeSize(victimAddressForScenarioB) > 0;
	//     }
	// }
	// Bytecode: 608060405234801561000f575f80fd5b506109818061001d5f395ff3fe60806040526004361061007e575f3560e01c80639c9e99851161004d5780639c9e9985146100dd578063b51c4f9614610107578063e10004dd14610143578063f8b2cb4f1461016d57610085565b80635a0e98531461008957806369a832f714610093578063734bf311146100bd57806380ebf041146100d357610085565b3661008557005b5f80fd5b6100916101a9565b005b34801561009e575f80fd5b506100a7610281565b6040516100b4919061056f565b60405180910390f35b3480156100c8575f80fd5b506100d16102a4565b005b6100db6103c0565b005b3480156100e8575f80fd5b506100f161042f565b6040516100fe91906105a2565b60405180910390f35b348015610112575f80fd5b5061012d600480360381019061012891906105e9565b6104be565b60405161013a919061062c565b60405180910390f35b34801561014e575f80fd5b506101576104de565b604051610164919061056f565b60405180910390f35b348015610178575f80fd5b50610193600480360381019061018e91906105e9565b610503565b6040516101a0919061062c565b60405180910390f35b5f346040516101b790610523565b6040518091039082f09050801580156101d2573d5f803e3d5ffd5b5090505f819050805f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff1662f55d9d306040518263ffffffff1660e01b81526004016102509190610665565b5f604051808303815f87803b158015610267575f80fd5b505af1158015610279573d5f803e3d5ffd5b505050505050565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f73ffffffffffffffffffffffffffffffffffffffff1660015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1603610333576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161032a906106fe565b60405180910390fd5b5f60015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690508073ffffffffffffffffffffffffffffffffffffffff1662f55d9d306040518263ffffffff1660e01b81526004016103909190610665565b5f604051808303815f87803b1580156103a7575f80fd5b505af11580156103b9573d5f803e3d5ffd5b5050505050565b5f346040516103ce90610523565b6040518091039082f09050801580156103e9573d5f803e3d5ffd5b5090508060015f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b5f8073ffffffffffffffffffffffffffffffffffffffff1660015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff160361048c575f90506104bb565b5f6104b760015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff166104be565b1190505b90565b5f8173ffffffffffffffffffffffffffffffffffffffff163b9050919050565b60015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f8173ffffffffffffffffffffffffffffffffffffffff16319050919050565b61022f8061071d83390190565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61055982610530565b9050919050565b6105698161054f565b82525050565b5f6020820190506105825f830184610560565b92915050565b5f8115159050919050565b61059c81610588565b82525050565b5f6020820190506105b55f830184610593565b92915050565b5f80fd5b6105c88161054f565b81146105d2575f80fd5b50565b5f813590506105e3816105bf565b92915050565b5f602082840312156105fe576105fd6105bb565b5b5f61060b848285016105d5565b91505092915050565b5f819050919050565b61062681610614565b82525050565b5f60208201905061063f5f83018461061d565b92915050565b5f61064f82610530565b9050919050565b61065f81610645565b82525050565b5f6020820190506106785f830184610656565b92915050565b5f82825260208201905092915050565b7f56696374696d20666f72205363656e6172696f2042206e6f74206372656174655f8201527f6420796574000000000000000000000000000000000000000000000000000000602082015250565b5f6106e860258361067e565b91506106f38261068e565b604082019050919050565b5f6020820190508181035f830152610715816106dc565b905091905056fe608060405261021e806100115f395ff3fe60806040526004361061002b575f3560e01c8062f55d9d146100365780634136aa351461005e57610032565b3661003257005b5f80fd5b348015610041575f80fd5b5061005c60048036038101906100579190610159565b610088565b005b348015610069575f80fd5b506100726100f3565b60405161007f919061019e565b60405180910390f35b5f4790508173ffffffffffffffffffffffffffffffffffffffff167f789ec66f21698ed1b990c0a8a8be99cf6f5fb8eb3826ee4ee9384870e8db25b1826040516100d291906101cf565b60405180910390a28173ffffffffffffffffffffffffffffffffffffffff16ff5b5f6001905090565b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610128826100ff565b9050919050565b6101388161011e565b8114610142575f80fd5b50565b5f813590506101538161012f565b92915050565b5f6020828403121561016e5761016d6100fb565b5b5f61017b84828501610145565b91505092915050565b5f8115159050919050565b61019881610184565b82525050565b5f6020820190506101b15f83018461018f565b92915050565b5f819050919050565b6101c9816101b7565b82525050565b5f6020820190506101e25f8301846101c0565b9291505056fea264697066735822122044d0802fb391d00f19322c84ab9d7131d07bee75d07c44254436419c7e0e809964736f6c63430008140033a2646970667358221220d683b3b16a0364301b40f4df517829a41da75df133e159e406032b323960085e64736f6c63430008140033
	// "80ebf041": "createVictimForScenarioB()",
	// "734bf311": "destroyVictimFromScenarioB()",
	// "f8b2cb4f": "getBalance(address)",
	// "b51c4f96": "getCodeSize(address)",
	// "9c9e9985": "isVictimBAlive()",
	// "5a0e9853": "testCreateAndDestroySameTx()",
	// "69a832f7": "victimAddressForScenarioA()",
	// "e10004dd": "victimAddressForScenarioB()"

	// Deploy contract
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(0)
	// Updated Bytecode for the new contract
	bytecode := common.FromHex("608060405234801561000f575f80fd5b506109818061001d5f395ff3fe60806040526004361061007e575f3560e01c80639c9e99851161004d5780639c9e9985146100dd578063b51c4f9614610107578063e10004dd14610143578063f8b2cb4f1461016d57610085565b80635a0e98531461008957806369a832f714610093578063734bf311146100bd57806380ebf041146100d357610085565b3661008557005b5f80fd5b6100916101a9565b005b34801561009e575f80fd5b506100a7610281565b6040516100b4919061056f565b60405180910390f35b3480156100c8575f80fd5b506100d16102a4565b005b6100db6103c0565b005b3480156100e8575f80fd5b506100f161042f565b6040516100fe91906105a2565b60405180910390f35b348015610112575f80fd5b5061012d600480360381019061012891906105e9565b6104be565b60405161013a919061062c565b60405180910390f35b34801561014e575f80fd5b506101576104de565b604051610164919061056f565b60405180910390f35b348015610178575f80fd5b50610193600480360381019061018e91906105e9565b610503565b6040516101a0919061062c565b60405180910390f35b5f346040516101b790610523565b6040518091039082f09050801580156101d2573d5f803e3d5ffd5b5090505f819050805f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff1662f55d9d306040518263ffffffff1660e01b81526004016102509190610665565b5f604051808303815f87803b158015610267575f80fd5b505af1158015610279573d5f803e3d5ffd5b505050505050565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f73ffffffffffffffffffffffffffffffffffffffff1660015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1603610333576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161032a906106fe565b60405180910390fd5b5f60015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690508073ffffffffffffffffffffffffffffffffffffffff1662f55d9d306040518263ffffffff1660e01b81526004016103909190610665565b5f604051808303815f87803b1580156103a7575f80fd5b505af11580156103b9573d5f803e3d5ffd5b5050505050565b5f346040516103ce90610523565b6040518091039082f09050801580156103e9573d5f803e3d5ffd5b5090508060015f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b5f8073ffffffffffffffffffffffffffffffffffffffff1660015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff160361048c575f90506104bb565b5f6104b760015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff166104be565b1190505b90565b5f8173ffffffffffffffffffffffffffffffffffffffff163b9050919050565b60015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f8173ffffffffffffffffffffffffffffffffffffffff16319050919050565b61022f8061071d83390190565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61055982610530565b9050919050565b6105698161054f565b82525050565b5f6020820190506105825f830184610560565b92915050565b5f8115159050919050565b61059c81610588565b82525050565b5f6020820190506105b55f830184610593565b92915050565b5f80fd5b6105c88161054f565b81146105d2575f80fd5b50565b5f813590506105e3816105bf565b92915050565b5f602082840312156105fe576105fd6105bb565b5b5f61060b848285016105d5565b91505092915050565b5f819050919050565b61062681610614565b82525050565b5f60208201905061063f5f83018461061d565b92915050565b5f61064f82610530565b9050919050565b61065f81610645565b82525050565b5f6020820190506106785f830184610656565b92915050565b5f82825260208201905092915050565b7f56696374696d20666f72205363656e6172696f2042206e6f74206372656174655f8201527f6420796574000000000000000000000000000000000000000000000000000000602082015250565b5f6106e860258361067e565b91506106f38261068e565b604082019050919050565b5f6020820190508181035f830152610715816106dc565b905091905056fe608060405261021e806100115f395ff3fe60806040526004361061002b575f3560e01c8062f55d9d146100365780634136aa351461005e57610032565b3661003257005b5f80fd5b348015610041575f80fd5b5061005c60048036038101906100579190610159565b610088565b005b348015610069575f80fd5b506100726100f3565b60405161007f919061019e565b60405180910390f35b5f4790508173ffffffffffffffffffffffffffffffffffffffff167f789ec66f21698ed1b990c0a8a8be99cf6f5fb8eb3826ee4ee9384870e8db25b1826040516100d291906101cf565b60405180910390a28173ffffffffffffffffffffffffffffffffffffffff16ff5b5f6001905090565b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610128826100ff565b9050919050565b6101388161011e565b8114610142575f80fd5b50565b5f813590506101538161012f565b92915050565b5f6020828403121561016e5761016d6100fb565b5b5f61017b84828501610145565b91505092915050565b5f8115159050919050565b61019881610184565b82525050565b5f6020820190506101b15f83018461018f565b92915050565b5f819050919050565b6101c9816101b7565b82525050565b5f6020820190506101e25f8301846101c0565b9291505056fea264697066735822122044d0802fb391d00f19322c84ab9d7131d07bee75d07c44254436419c7e0e809964736f6c63430008140033a2646970667358221220d683b3b16a0364301b40f4df517829a41da75df133e159e406032b323960085e64736f6c63430008140033")

	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    1, // Assuming nonce starts at 1 after setup
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err := suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle contract deployment tx")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	var res types.MsgEthereumTxResponse
	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode deployment result data")
	suite.Require().Equal(res.VmError, "", "contract deployment failed")
	suite.Require().NotEmpty(res.Ret, "contract deployment returned empty address")
	// contractAddr := common.BytesToAddress(res.Ret) // Get the deployed contract address
	contractAddr := crypto.CreateAddress(suite.from, 1)

	// fmt.Println("Contract Address:", contractAddr.Hex())

	//////////////////////////////////////////////////////////////////////
	// Scenario A: Construct and destroy in the same transaction
	//////////////////////////////////////////////////////////////////////

	// Call testCreateAndDestroySameTx()
	bytecode = common.FromHex("0x5a0e9853")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	// Call victimAddressForScenarioA()
	bytecode = common.FromHex("0x69a832f7")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    3,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	victimAddressForScenarioA := common.BytesToAddress(res.Ret)

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	// Call getCodeSize()
	bytecode = common.FromHex("0xb51c4f96")
	argValue := common.FromHex(victimAddressForScenarioA.Hex())
	input := createInputWithMethidAndValue(bytecode, argValue)

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    4,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    input,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// Check the code size of the victim contract
	codeSize := new(big.Int).SetBytes(res.Ret)
	suite.Require().Equal(uint64(0), codeSize.Uint64(), "Victim contract should be destroyed in Scenario A")

	//TODO: Check the balance of the tester contract

	// // Check the balance of the victim contract
	// balance := new(big.Int).SetBytes(res.Ret)
	// suite.Require().Equal(uint64(0), balance.Uint64(), "Victim contract should have no balance in Scenario A")
	// // Check the balance of the tester contract
	// testerBalance := new(big.Int).SetBytes(res.Ret)
	// suite.Require().Equal(uint64(0), testerBalance.Uint64(), "Tester contract should have no balance in Scenario A")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	//////////////////////////////////////////////////////////////////////
	// Scenario B: Destroy Tx after construct Tx
	//////////////////////////////////////////////////////////////////////
	// Call createVictimForScenarioB()
	bytecode = common.FromHex("0x80ebf041")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    5,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// Call victimAddressForScenarioB()
	bytecode = common.FromHex("0xe10004dd")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    6,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	// Call destroyVictimFromScenarioB()
	bytecode = common.FromHex("0x734bf311")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    7,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	// Call isVictimBAlive()
	bytecode = common.FromHex("0x9c9e9985")

	ethTxParams = &types.EvmTxArgs{
		ChainID:  suite.chainID,
		Nonce:    8,
		To:       &contractAddr,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = types.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, err = suite.handler(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	err = proto.Unmarshal(result.Data, &res)
	suite.Require().NoError(err, "failed to decode result data")
	suite.Require().Equal(res.VmError, "", "failed to handle eth tx msg")

	// Check the code size of the victim contract
	isAlive := new(big.Int).SetBytes(res.Ret)
	suite.Require().True(isAlive.Uint64() > 0, "Victim contract should not be destroyed in Scenario B")
	// suite.Require().Equal(uint64(1), isAlive.Uint64(), "Victim contract should not be destroyed in Scenario B")

	//TODO: Check the balance of the tester contract

	// // Check the balance of the victim contract
	// balance := new(big.Int).SetBytes(res.Ret)
	// suite.Require().Equal(uint64(0), balance.Uint64(), "Victim contract should have no balance in Scenario B")
	// // Check the balance of the tester contract
	// testerBalance := new(big.Int).SetBytes(res.Ret)
	// suite.Require().Equal(uint64(0), testerBalance.Uint64(), "Tester contract should have no balance in Scenario B")

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

}
