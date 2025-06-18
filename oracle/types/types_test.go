package types_test

import (
	"flag"
	"strconv"
	"testing"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/bytes"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type TypesTestSuite struct {
	suite.Suite
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) SetupSuite() {
	// Initialize logger for the test suite
	log.InitLogger()

	// Load config once for the entire test suite
	// Use a temporary directory for daemon data to avoid conflicts
	tempDir := suite.T().TempDir()
	flag.Set("daemon-dir", tempDir)
	err := config.LoadConfig()
	suite.Require().NoError(err, "Failed to load config for test suite")
}

func (suite *TypesTestSuite) SetupTest() {
	config.Config.SetAddress("my_address")
}

func (suite *TypesTestSuite) TestMakeJobs_FromOracleRequestDoc() {
	testCases := []struct {
		name        string
		doc         *oracletypes.OracleRequestDoc
		expectedLen int
		expectedURL string
	}{
		{
			name: "Success - address in list",
			doc: &oracletypes.OracleRequestDoc{
				RequestId:   1,
				AccountList: []string{"another_address", "my_address"},
				Endpoints: []*oracletypes.OracleEndpoint{
					{Url: "http://url1.com", ParseRule: "path1"},
					{Url: "http://url2.com", ParseRule: "path2"},
				},
				Nonce:  10,
				Period: 60,
				Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
			},
			expectedLen: 1,
			expectedURL: "http://url1.com", // myIndex=1, so (myIndex+0)%2 = 1 -> url2 in original logic, but now it is (myIndex+1)%len
		},
		{
			name: "Success - address not in list",
			doc: &oracletypes.OracleRequestDoc{
				RequestId:   2,
				AccountList: []string{"addr1", "addr2"},
				Endpoints: []*oracletypes.OracleEndpoint{
					{Url: "http://url1.com", ParseRule: "path1"},
				},
			},
			expectedLen: 1, // It still creates a job, which would be filtered by the daemon
			expectedURL: "http://url1.com",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			jobs := types.MakeJobs(tc.doc)
			suite.Require().Len(jobs, tc.expectedLen)
			if tc.expectedLen > 0 {
				suite.Equal(tc.doc.RequestId, jobs[0].ID)
				suite.Equal(tc.expectedURL, jobs[0].URL)
				suite.Equal(tc.doc.Nonce, jobs[0].Nonce)
				suite.Equal(time.Duration(tc.doc.Period)*time.Second, jobs[0].Delay)
			}
		})
	}

	suite.Run("Panic on empty endpoints", func() {
		doc := &oracletypes.OracleRequestDoc{
			RequestId:   3,
			AccountList: []string{"my_address"},
			Endpoints:   []*oracletypes.OracleEndpoint{},
		}
		suite.Panics(func() {
			types.MakeJobs(doc)
		})
	})
}

func (suite *TypesTestSuite) TestMakeJobs_FromResultEventNewBlock() {
	event := coretypes.ResultEvent{
		Query: "tm.event='NewBlock'",
		Data:  tmtypes.EventDataNewBlock{},
		Events: map[string][]string{
			"complete_oracle_data_set.request_id": {"101", "102"},
			"complete_oracle_data_set.nonce":      {"201", "202"},
			"some_other.event":                    {"value"},
		},
	}

	jobs := types.MakeJobs(event)
	suite.Require().Len(jobs, 2)

	suite.Equal(uint64(101), jobs[0].ID)
	suite.Equal(uint64(201), jobs[0].Nonce)
	suite.Equal(types.Complete, jobs[0].Type)

	suite.Equal(uint64(102), jobs[1].ID)
	suite.Equal(uint64(202), jobs[1].Nonce)
	suite.Equal(types.Complete, jobs[1].Type)
}

func (suite *TypesTestSuite) TestMakeJobs_FromResultEventTx() {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	txBuilder := encCfg.TxConfig.NewTxBuilder()

	msg := &oracletypes.MsgRegisterOracleRequestDoc{
		RequestDoc: oracletypes.OracleRequestDoc{
			AccountList: []string{"my_address"},
			Endpoints:   []*oracletypes.OracleEndpoint{{Url: "http://test.com", ParseRule: "price"}},
			Period:      30,
		},
	}
	suite.NoError(txBuilder.SetMsgs(msg))
	tx := txBuilder.GetTx()
	txBytes, err := encCfg.TxConfig.TxEncoder()(tx)
	suite.NoError(err)

	event := coretypes.ResultEvent{
		Query: "tm.event='Tx'",
		Data: tmtypes.EventDataTx{
			TxResult: abcitypes.TxResult{
				Height: 1,
				Index:  0,
				Tx:     txBytes,
			},
		},
		Events: map[string][]string{
			"register_oracle_request_doc.request_id": {strconv.FormatUint(123, 10)},
		},
	}

	jobs := types.MakeJobs(event)
	suite.Require().Len(jobs, 1)
	suite.Equal(uint64(123), jobs[0].ID)
	suite.Equal("http://test.com", jobs[0].URL)
	suite.Equal("price", jobs[0].Path)
	suite.Equal(types.Register, jobs[0].Type)
	suite.Equal(uint64(0), jobs[0].Nonce)
	suite.Equal(30*time.Second, jobs[0].Delay)
}

func (suite *TypesTestSuite) TestMakeJobs_UnsupportedType() {
	unsupportedEvent := bytes.HexBytes("unsupported")
	jobs := types.MakeJobs(unsupportedEvent)
	suite.Empty(jobs)
}
