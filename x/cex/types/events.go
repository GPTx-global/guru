package types

// cex module event types
const (
	// event types
	EventTypeSwap                   = ModuleName + "_swap"
	EventTypeRegisterReserveAccount = ModuleName + "_register_reserve_account"
	EventTypeRegisterAdmin          = ModuleName + "_register_admin"
	EventTypeUpdateRate             = ModuleName + "_update_rate"
	EventTypeChangeModerator        = ModuleName + "_change_moderator_address"

	// event attributes
	AttributeKeyAddress   = "address"
	AttributeKeyModerator = "moderator"
	AttributeKeyAdmin     = "admin"
	AttributeKeyPairDenom = "pair_denom"
	AttributeKeyRate      = "currency_rate"
	AttributeKeyAmount    = "amount"
)
