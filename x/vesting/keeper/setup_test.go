package keeper_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	evm "github.com/GPTx-global/guru/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/x/vesting/types"
)

var (
	contract  common.Address
	contract2 common.Address
)

var (
	erc20Name     = "Coin Token"
	erc20Symbol   = "CTKN"
	erc20Name2    = "Coin Token 2"
	erc20Symbol2  = "CTKN2"
	erc20Decimals = uint8(18)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	app            *app.Guru
	queryClientEvm evm.QueryClient
	queryClient    types.QueryClient
	address        common.Address
	consAddress    sdk.ConsAddress
	validator      stakingtypes.Validator
	clientCtx      client.Context
	ethSigner      ethtypes.Signer
	priv           cryptotypes.PrivKey
	signer         keyring.Signer
}

var s *KeeperTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}
