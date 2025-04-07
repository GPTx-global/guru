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

// DoSetupTest setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *EvmTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())
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

	coins := sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdkmath.NewInt(100000000000000)))
	genesisState := app.NewTestGenesisState(suite.app.AppCodec())
	b32address := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
	balances := []banktypes.Balance{
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
	bankGenesis.Supply = bankGenesis.Supply.Add(coins...).Add(coins...)
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
	validator, err := stakingtypes.NewValidator(valAddr, priv.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)

	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	suite.app.StakingKeeper.SetValidator(suite.ctx, validator)

	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
	suite.handler = evm.NewHandler(suite.app.EvmKeeper)
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
	//             // Use MCOPY
	//             // Args: desc offset, src offset, bytes
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

	// keccak-256("copyMemory(bytes)")
	bytecode = common.FromHex("0xc83aa2f3")

	argValue := common.FromHex("0xff")
	argLength := big.NewInt(int64(len(argValue)))
	argOffset := big.NewInt(32)

	offsetBytes := make([]byte, 32)
	argOffset.FillBytes(offsetBytes)

	lengthBytes := make([]byte, 32)
	argLength.FillBytes(lengthBytes)

	valueBytes := make([]byte, 32)
	copy(valueBytes, argValue)

	input := append(bytecode, offsetBytes...)
	input = append(input, lengthBytes...)
	input = append(input, valueBytes...)

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
	// keccak-256("copyMemory(bytes)")
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
