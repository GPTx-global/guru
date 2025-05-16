package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/GPTx-global/guru/x/cex/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (k Keeper) AddNewAdmin(ctx sdk.Context, admin types.Admin) error {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	bz := adminStore.Get([]byte(admin.Address))
	if bz != nil {
		return fmt.Errorf("admin is already registered")
	}

	dataBytes := k.cdc.MustMarshal(&admin.Exchanges)

	adminStore.Set([]byte(admin.Address), dataBytes)

	return nil
}

func (k Keeper) SetAdmin(ctx sdk.Context, newAdminAddr string, exchangeId math.Int) error {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	exchanges := types.AdminExchanges{}

	// get the list of admin exchanges
	bz := adminStore.Get([]byte(newAdminAddr))
	if bz != nil {
		k.cdc.MustUnmarshal(bz, &exchanges)
	}

	// check if exchange already belongs to admin
	for _, id := range exchanges.Ids {
		if id.Equal(exchangeId) {
			return fmt.Errorf("admin is already set")
		}
	}

	// register new exchange to the admin
	exchanges.Ids = append(exchanges.Ids, exchangeId)

	// update the KV store
	dataBytes := k.cdc.MustMarshal(&exchanges)
	adminStore.Set([]byte(newAdminAddr), dataBytes)

	return nil
}

func (k Keeper) DeleteAdmin(ctx sdk.Context, adminAddr string) {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	adminStore.Delete([]byte(adminAddr))
}

func (k Keeper) IsAdmin(ctx sdk.Context, adminAddr string) bool {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	bz := adminStore.Get([]byte(adminAddr))
	return bz != nil
}

func (k Keeper) IsAdminOf(ctx sdk.Context, adminAddr string, exchangeId math.Int) bool {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	bz := adminStore.Get([]byte(adminAddr))
	if bz == nil {
		return false
	}

	exchanges := types.AdminExchanges{}
	k.cdc.MustUnmarshal(bz, &exchanges)

	// check if exchange belongs to admin
	for _, id := range exchanges.Ids {
		if id.Equal(exchangeId) {
			return true
		}
	}

	return false
}

func (k Keeper) GetPaginatedAdmins(ctx sdk.Context, pagination *query.PageRequest) ([]types.Admin, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	adminStore := prefix.NewStore(store, types.KeyAdmins)

	admins := []types.Admin{}

	pageRes, err := query.Paginate(adminStore, pagination, func(key, value []byte) error {
		var exchanges types.AdminExchanges
		k.cdc.MustUnmarshal(value, &exchanges)

		// add the admin to the list
		admins = append(admins, types.Admin{Address: string(key), Exchanges: exchanges})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return admins, pageRes, nil
}
