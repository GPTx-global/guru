package subscribe

import (
	"context"
	"flag"
	"testing"

	"github.com/GPTx-global/guru/oracle/config"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/stretchr/testify/suite"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type SubscribeManagerTestSuite struct {
	suite.Suite
	manager *SubscribeManager
}

func TestSubscribeManagerTestSuite(t *testing.T) {
	suite.Run(t, new(SubscribeManagerTestSuite))
}

func (suite *SubscribeManagerTestSuite) SetupSuite() {
	tempDir := suite.T().TempDir()
	flag.Set("daemon-dir", tempDir)
	config.Load()
}

func (suite *SubscribeManagerTestSuite) SetupTest() {
	suite.manager = NewSubscribeManager(context.Background())

}

func (suite *SubscribeManagerTestSuite) TestNewSubscribeManager() {
	suite.NotNil(suite.manager)
}

func (suite *SubscribeManagerTestSuite) TestFilterAccount() {
	testCases := []struct {
		name          string
		eventAccounts string
		expected      bool
	}{
		{
			"address in list",
			"addr1,my_test_address,addr2",
			true,
		},
		{
			"address not in list",
			"addr1,addr2,addr3",
			false,
		},
		{
			"empty account list",
			"",
			false,
		},
		{
			"address as substring",
			"my_test_address_extra,addr2",
			true, // Note: strings.Contains behavior
		},
		{
			"exact match needed and present",
			"addr1,my_test_address,addr2",
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			event := coretypes.ResultEvent{
				Events: map[string][]string{
					"test_prefix." + oracletypes.AttributeKeyAccountList: {tc.eventAccounts},
				},
			}
			result := suite.manager.filterAccount(event, "test_prefix")
			suite.Equal(tc.expected, result)
		})
	}
}
