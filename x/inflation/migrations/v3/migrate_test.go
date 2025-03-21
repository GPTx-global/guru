package v3_test

import (
	"testing"

	v3 "github.com/GPTx-global/guru/x/inflation/migrations/v3"
	"github.com/GPTx-global/guru/x/inflation/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(types.ModuleName)
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	store.Set(v3.KeyPrefixEpochMintProvision, []byte{0x01})
	epochMintProvision := store.Get(v3.KeyPrefixEpochMintProvision)
	require.Equal(t, epochMintProvision, []byte{0x01})

	require.NoError(t, v3.MigrateStore(store))
	require.False(t, store.Has(v3.KeyPrefixEpochMintProvision))
}
