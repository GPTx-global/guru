package types

// cex module event types
const (
	// event types
	EventTypeSwap             = ModuleName + "_swap"
	EventTypeRegisterAdmin    = ModuleName + "_register_admin"
	EventTypeRemoveAdmin      = ModuleName + "_remove_admin"
	EventTypeRegisterExchange = ModuleName + "_register_exchange"
	EventTypeUpdateExchange   = ModuleName + "_update_exchange"
	EventTypeChangeModerator  = ModuleName + "_change_moderator_address"

	// event attributes
	AttributeKeyAddress    = "address"
	AttributeKeyModerator  = "moderator"
	AttributeKeyAdmin      = "admin"
	AttributeKeyExchangeId = "exchange_id"
	AttributeKeyRate       = "rate"
	AttributeKeyAttributes = "attributes"
	AttributeKeyAmount     = "amount"
)
