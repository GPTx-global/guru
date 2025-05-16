package types

const (
	// module name
	ModuleName = "cex"

	// StoreKey is the default store key for the module
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

// KV Store key prefix bytes
const (
	prefixModeratorAddress = iota + 1
	prefixExchanges
	prefixAdmins
	prefixNextExchangeId
)

// KV Store key prefixes
var (
	KeyModeratorAddress = []byte{prefixModeratorAddress}
	KeyExchanges        = []byte{prefixExchanges}
	KeyAdmins           = []byte{prefixAdmins}
	KeyNextExchangeId   = []byte{prefixNextExchangeId}
)

// default keys
var (
	KeyExchangeId             = "id"
	KeyExchangeReserveAddress = "reserve_address"
	KeyExchangeBaseIBC        = "base_coin_ibc_denom"
	KeyExchangeQuoteIBC       = "quote_coin_ibc_denom"
	KeyExchangeBaseShort      = "base_coin_short_denom"
	KeyExchangeQuoteShort     = "quote_coin_short_denom"
	KeyExchangeRate           = "rate"
	KeyExchangeStatus         = "status"
	KeyExchangeFee            = "fee"
	KeyExchangeAccumulatedFee = "accumulated_fee"
	RequiredKeysExchange      = []*string{&KeyExchangeReserveAddress, &KeyExchangeBaseIBC, &KeyExchangeQuoteIBC, &KeyExchangeBaseShort, &KeyExchangeQuoteShort, &KeyExchangeRate, &KeyExchangeFee}
)

// func GetCoinKey(ibcDenom string) []byte {
// 	return append(KeyCoinPair, []byte(pairDenom)...)
// }
