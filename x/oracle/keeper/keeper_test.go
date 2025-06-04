package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/GPTx-global/guru/x/oracle/types"
)

// setupKeeper creates a new Keeper instance and context for testing
func setupKeeper(t *testing.T) (*Keeper, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	cdc := codec.NewProtoCodec(nil)

	keeper := NewKeeper(
		cdc,
		storeKey,
	)

	return keeper, ctx
}

// TestSetAndGetOracleRequestDocCount tests the setting and getting of oracle request document count
func TestSetAndGetOracleRequestDocCount(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	// Initial count should be 0
	initialCount := keeper.GetOracleRequestDocCount(ctx)
	assert.Equal(t, uint64(0), initialCount)

	// Set count
	testCount := uint64(42)
	keeper.SetOracleRequestDocCount(ctx, testCount)

	// Verify the set count
	retrievedCount := keeper.GetOracleRequestDocCount(ctx)
	assert.Equal(t, testCount, retrievedCount)
}

// TestSetAndGetOracleRequestDoc tests the setting and getting of oracle request documents
func TestSetAndGetOracleRequestDoc(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	// Create test document
	doc := types.OracleRequestDoc{
		RequestId: 1,
		// Add other required fields
	}

	// Store document
	keeper.SetOracleRequestDoc(ctx, doc)

	// Retrieve document
	retrievedDoc, err := keeper.GetOracleRequestDoc(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, doc.RequestId, retrievedDoc.RequestId)

	// Test retrieval of non-existent document
	_, err = keeper.GetOracleRequestDoc(ctx, 999)
	assert.Error(t, err)
}

// TestSetAndGetModeratorAddress tests the setting and getting of moderator address
func TestSetAndGetModeratorAddress(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	// Initial address should be empty string
	initialAddress := keeper.GetModeratorAddress(ctx)
	assert.Equal(t, "", initialAddress)

	// Set address
	testAddress := "cosmos1testaddress"
	keeper.SetModeratorAddress(ctx, testAddress)

	// Verify the set address
	retrievedAddress := keeper.GetModeratorAddress(ctx)
	assert.Equal(t, testAddress, retrievedAddress)
}
