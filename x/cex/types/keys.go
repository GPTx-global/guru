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
	KeyExchangeCoinAIBCDenom  = "coin_a_ibc_denom"
	KeyExchangeCoinBIBCDenom  = "coin_b_ibc_denom"
	KeyExchangeCoinAShort     = "coin_a_short_denom"
	KeyExchangeCoinBShort     = "coin_b_short_denom"
	KeyExchangeCoinAPort      = "coin_a_port"
	KeyExchangeCoinBPort      = "coin_b_port"
	KeyExchangeCoinAChannel   = "coin_a_channel"
	KeyExchangeCoinBChannel   = "coin_b_channel"
	KeyExchangeAtoBRate       = "a_to_b_rate"
	KeyExchangeBtoARate       = "b_to_a_rate"
	KeyExchangeStatus         = "status"
	KeyExchangeFee            = "fee"
	KeyExchangeAccumulatedFee = "accumulated_fee"
	RequiredKeysExchange      = []*string{
		&KeyExchangeReserveAddress,
		&KeyExchangeCoinAIBCDenom,
		&KeyExchangeCoinBIBCDenom,
		&KeyExchangeCoinAShort,
		&KeyExchangeCoinBShort,
		&KeyExchangeCoinAPort,
		&KeyExchangeCoinBPort,
		&KeyExchangeCoinAChannel,
		&KeyExchangeCoinBChannel,
		&KeyExchangeAtoBRate,
		&KeyExchangeBtoARate,
		&KeyExchangeFee,
	}
)

// func GetCoinKey(ibcDenom string) []byte {
// 	return append(KeyCoinPair, []byte(pairDenom)...)
// }
