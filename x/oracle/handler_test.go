package oracle

import (
	"testing"

	"github.com/GPTx-global/guru/x/oracle/keeper"
	"github.com/GPTx-global/guru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func setupHandlerTest(t *testing.T) (sdk.Context, *keeper.Keeper) {
	ctx, k := setupTest(t)
	k.SetModeratorAddress(ctx, "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft")
	return ctx, k
}

func TestNewHandler(t *testing.T) {
	ctx, k := setupHandlerTest(t)
	handler := NewHandler(k)

	tests := []struct {
		name    string
		msg     sdk.Msg
		wantErr bool
	}{
		{
			name: "register oracle request doc",
			msg: &types.MsgRegisterOracleRequestDoc{
				ModeratorAddress: "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				RequestDoc: types.OracleRequestDoc{
					OracleType:      types.OracleType_ORACLE_TYPE_CRYPTO,
					Name:            "BTC/USD Price",
					Description:     "BTC/USD price oracle",
					Period:          60,
					AccountList:     []string{"guru1z0jjc8gx8quevavj2yqwu2a8tm6dy9fsw6a2wz", "guru172m263wcg98cph8e32tv0xky94eq4r4wzhwgaz"},
					Quorum:          2,
					Endpoints:       []*types.OracleEndpoint{{Url: "https://api.coinbase.com/v2/prices/BTC-USD/spot", ParseRule: "data.amount"}},
					AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
					Status:          types.RequestStatus_REQUEST_STATUS_ENABLED,
				},
			},
			wantErr: false,
		},
		{
			name: "update oracle request doc",
			msg: &types.MsgUpdateOracleRequestDoc{
				ModeratorAddress: "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				RequestDoc: types.OracleRequestDoc{
					RequestId:       1,
					OracleType:      types.OracleType_ORACLE_TYPE_CRYPTO,
					Name:            "BTC/USD Price",
					Description:     "BTC/USD price oracle",
					Period:          120,
					AccountList:     []string{"guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft"},
					Quorum:          2,
					Endpoints:       []*types.OracleEndpoint{{Url: "https://api.coinbase.com/v2/prices/BTC-USD/spot", ParseRule: "data.amount"}},
					AggregationRule: types.AggregationRule_AGGREGATION_RULE_AVG,
					Status:          types.RequestStatus_REQUEST_STATUS_PAUSED,
				},
				Reason: "Update period and status",
			},
			wantErr: false,
		},
		{
			name: "submit oracle data",
			msg: &types.MsgSubmitOracleData{
				AuthorityAddress: "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				DataSet: &types.SubmitDataSet{
					RequestId: 1,
					Nonce:     1,
					RawData:   "50000",
					Provider:  "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				},
			},
			wantErr: false,
		},
		{
			name: "update moderator address",
			msg: &types.MsgUpdateModeratorAddress{
				ModeratorAddress:    "guru1h9y8h0rh6tqxrj045fyvarnnyyxdg07693zkft",
				NewModeratorAddress: "guru1z0jjc8gx8quevavj2yqwu2a8tm6dy9fsw6a2wz",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler(ctx, tt.msg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
