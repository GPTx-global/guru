package oracle

import (
	"testing"

	"github.com/GPTx-global/guru/x/oracle/keeper"
	"github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

func setupTest(t *testing.T) (sdk.Context, *keeper.Keeper) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("guru", "gurupub")

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	return ctx, k
}

func TestInitGenesis(t *testing.T) {
	ctx, k := setupTest(t)

	tests := []struct {
		name     string
		genesis  types.GenesisState
		expPanic bool
	}{
		{
			name: "1. valid genesis state",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:       1,
						OracleType:      types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
						Name:            "Test Oracle",
						Description:     "Test Description",
						Period:          60,
						AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
						Quorum:          1,
						Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: false,
		},
		{
			name: "2. empty moderator address",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress: "",
			},
			expPanic: true,
		},
		{
			name: "3. invalid oracle request doc",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:       1,
						OracleType:      types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
						Name:            "", // Empty name should cause validation error
						Description:     "Test Description",
						Period:          60,
						AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
						Quorum:          1,
						Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: true,
		},
		{
			name: "4. unspecified oracle type",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:       1,
						OracleType:      types.OracleType_ORACLE_TYPE_UNSPECIFIED,
						Name:            "Test Oracle",
						Description:     "Test Description",
						Period:          60,
						AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
						Quorum:          1,
						Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: true,
		},
		{
			name: "5. empty endpoints",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:       1,
						OracleType:      types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
						Name:            "Test Oracle",
						Description:     "Test Description",
						Period:          60,
						AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
						Quorum:          1,
						Endpoints:       []*types.OracleEndpoint{},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: true,
		},
		{
			name: "6. invalid account address",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:       1,
						OracleType:      types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
						Name:            "Test Oracle",
						Description:     "Test Description",
						Period:          60,
						AccountList:     []string{"invalid-address"},
						Quorum:          1,
						Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: true,
		},
		{
			name: "7. quorum greater than account list length",
			genesis: types.GenesisState{
				Params: types.Params{
					EnableOracle: true,
				},
				ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				OracleRequestDocCount: 1,
				OracleRequestDocs: []types.OracleRequestDoc{
					{
						RequestId:   1,
						OracleType:  types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
						Name:        "Test Oracle",
						Description: "Test Description",
						Period:      60,
						AccountList: []string{
							"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
							"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
						}, // 2개의 계정
						Quorum:          3, // 3개의 quorum (계정 수보다 큼)
						Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
						AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
						Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
					},
				},
			},
			expPanic: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expPanic {
				require.Panics(t, func() {
					InitGenesis(ctx, *k, tc.genesis)
				})
			} else {
				require.NotPanics(t, func() {
					InitGenesis(ctx, *k, tc.genesis)
				})
			}

			// for _, doc := range tc.genesis.OracleRequestDocs {
			// 	found, err := k.GetOracleRequestDoc(ctx, doc.RequestId)
			// 	require.NoError(t, err)
			// 	require.Equal(t, doc, *found)
			// }
		})
	}
}

func TestExportGenesis(t *testing.T) {
	ctx, k := setupTest(t)

	// Initialize state with test data
	genesis := types.GenesisState{
		Params: types.Params{
			EnableOracle:          true,
			MinSubmitPerWindow:    sdk.MustNewDecFromStr("0.5"),
			SlashFractionDowntime: sdk.MustNewDecFromStr("0.0001"),
		},
		ModeratorAddress:      "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
		OracleRequestDocCount: 1,
		OracleRequestDocs: []types.OracleRequestDoc{
			{
				RequestId:       1,
				OracleType:      types.OracleType_ORACLE_TYPE_MIN_GAS_PRICE,
				Name:            "Test Oracle",
				Description:     "Test Description",
				Period:          60,
				AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
				Quorum:          1,
				Endpoints:       []*types.OracleEndpoint{{Url: "http://test.com", ParseRule: "test"}},
				AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
				Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
			},
		},
	}

	InitGenesis(ctx, *k, genesis)

	// Export genesis state
	exported := ExportGenesis(ctx, *k)

	// Verify exported state matches initialized state
	require.Equal(t, genesis.Params, exported.Params)
	require.Equal(t, genesis.ModeratorAddress, exported.ModeratorAddress)
	require.Equal(t, len(genesis.OracleRequestDocs), len(exported.OracleRequestDocs))
	for i, doc := range genesis.OracleRequestDocs {
		require.Equal(t, doc, exported.OracleRequestDocs[i])
	}
}
