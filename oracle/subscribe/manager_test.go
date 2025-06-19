package subscribe

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	EventTypeRegisterOracleRequestDoc = "register_oracle_request_doc"
	EventTypeUpdateOracleRequestDoc   = "update_oracle_request_doc"
	EventTypeCompleteDataSet          = "complete_oracle_data_set"
	AttributeKeyRequestID             = "request_id"
	AttributeKeyAccountList           = "account_list"
)

type SubscribeManagerSuite struct {
	suite.Suite
	sm           *SubscribeManager
	ctx          context.Context
	cancel       context.CancelFunc
	testAddr     sdk.AccAddress
	registerChan chan coretypes.ResultEvent
	updateChan   chan coretypes.ResultEvent
	completeChan chan coretypes.ResultEvent
}

func (s *SubscribeManagerSuite) SetupSuite() {
	log.InitLogger()
	config.SetForTesting("test-chain", "", "test", os.TempDir(), keyring.BackendTest, "", 0, 3)

	kr := config.Keyring()
	_, _, err := kr.NewMnemonic(config.KeyName(), keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err, "failed to create test key")

	s.testAddr = config.Address()
}

func (s *SubscribeManagerSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.sm = NewSubscribeManager(s.ctx)

	// Manually create channels and inject them into the manager
	s.registerChan = make(chan coretypes.ResultEvent, 1)
	s.updateChan = make(chan coretypes.ResultEvent, 1)
	s.completeChan = make(chan coretypes.ResultEvent, 1)
	s.sm.subscriptions[registerMsg] = s.registerChan
	s.sm.subscriptions[updateMsg] = s.updateChan
	s.sm.subscriptions[completeMsg] = s.completeChan
}

func (s *SubscribeManagerSuite) TearDownTest() {
	s.cancel()
}

func TestSubscribeManagerSuite(t *testing.T) {
	suite.Run(t, new(SubscribeManagerSuite))
}

func (s *SubscribeManagerSuite) TestFilterAccount() {
	otherAddr := "cosmos1nhtc6ysy8b7f5h9nrzchcrx82z8v50a8z5e3g7"

	testCases := []struct {
		name          string
		eventAccounts string
		prefix        string
		shouldContain bool
	}{
		{"AccountInList", fmt.Sprintf("%s,%s", s.testAddr.String(), otherAddr), EventTypeRegisterOracleRequestDoc, true},
		{"AccountNotInList", otherAddr, EventTypeRegisterOracleRequestDoc, false},
		{"AccountIsTheOnlyOne", s.testAddr.String(), EventTypeRegisterOracleRequestDoc, true},
		{"EmptyList", "", EventTypeRegisterOracleRequestDoc, false},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			event := coretypes.ResultEvent{
				Events: map[string][]string{
					tc.prefix + "." + AttributeKeyAccountList: {tc.eventAccounts},
				},
			}
			if _, ok := event.Events[tc.prefix+"."+AttributeKeyAccountList]; !ok {
				event.Events[tc.prefix+"."+AttributeKeyAccountList] = []string{""}
			}

			result := s.sm.filterAccount(event, tc.prefix)
			s.Require().Equal(tc.shouldContain, result)
		})
	}
}

func (s *SubscribeManagerSuite) TestSubscribe_RegisterEventSuccess() {
	var wg sync.WaitGroup
	var receivedJobs []*types.Job

	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedJobs = s.sm.Subscribe()
	}()

	registerEvent := coretypes.ResultEvent{
		Data: tmtypes.EventDataTx{},
		Events: map[string][]string{
			EventTypeRegisterOracleRequestDoc + "." + AttributeKeyRequestID:   {"1"},
			EventTypeRegisterOracleRequestDoc + "." + "url":                   {"http://test.com"},
			EventTypeRegisterOracleRequestDoc + "." + "path":                  {"price"},
			EventTypeRegisterOracleRequestDoc + "." + "nonce":                 {"0"},
			EventTypeRegisterOracleRequestDoc + "." + AttributeKeyAccountList: {s.testAddr.String()},
		},
	}
	s.registerChan <- registerEvent
	wg.Wait()

	s.Require().Nil(receivedJobs)
}

func (s *SubscribeManagerSuite) TestSubscribe_RegisterEventFiltered() {
	var wg sync.WaitGroup
	var receivedJobs []*types.Job

	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedJobs = s.sm.Subscribe()
	}()

	filteredEvent := coretypes.ResultEvent{
		Data: tmtypes.EventDataTx{},
		Events: map[string][]string{
			EventTypeRegisterOracleRequestDoc + "." + AttributeKeyRequestID:   {"2"},
			EventTypeRegisterOracleRequestDoc + "." + AttributeKeyAccountList: {"cosmos1nhtc6ysy8b7f5h9nrzchcrx82z8v50a8z5e3g7"},
		},
	}
	s.registerChan <- filteredEvent
	wg.Wait()

	s.Require().Nil(receivedJobs, "Jobs should be nil for filtered events")
}

func (s *SubscribeManagerSuite) TestSubscribe_UpdateEvent() {
	var wg sync.WaitGroup
	var receivedJobs []*types.Job

	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedJobs = s.sm.Subscribe()
	}()

	updateEvent := coretypes.ResultEvent{
		Data: tmtypes.EventDataTx{},
		Events: map[string][]string{
			EventTypeUpdateOracleRequestDoc + "." + AttributeKeyRequestID: {"3"},
			EventTypeUpdateOracleRequestDoc + "." + "url":                 {"http://new-test.com"},
			EventTypeUpdateOracleRequestDoc + "." + "path":                {"new_price"},
		},
	}
	s.updateChan <- updateEvent
	wg.Wait()

	s.Require().Nil(receivedJobs)
}

func (s *SubscribeManagerSuite) TestSubscribe_CompleteEvent() {
	var wg sync.WaitGroup
	var receivedJobs []*types.Job

	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedJobs = s.sm.Subscribe()
	}()

	completeEvent := coretypes.ResultEvent{
		Data: tmtypes.EventDataNewBlock{},
		Events: map[string][]string{
			EventTypeCompleteDataSet + "." + AttributeKeyRequestID: {"4"},
			EventTypeCompleteDataSet + "." + "nonce":               {"10"},
		},
	}
	s.completeChan <- completeEvent
	wg.Wait()

	s.Require().NotNil(receivedJobs)
	s.Require().Len(receivedJobs, 1)
	s.Require().Equal(types.Complete, receivedJobs[0].Type)
}

func (s *SubscribeManagerSuite) TestSubscribe_ContextCanceled() {
	var wg sync.WaitGroup
	var receivedJobs []*types.Job

	ctx, cancel := context.WithCancel(context.Background())
	sm := NewSubscribeManager(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedJobs = sm.Subscribe()
	}()

	cancel()
	wg.Wait()

	s.Require().Nil(receivedJobs, "Subscribe should return nil when context is canceled")
}
