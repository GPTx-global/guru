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
	prefixRatemeter
	prefixAddressRequestCount
)

// KV Store key prefixes
var (
	KeyModeratorAddress    = []byte{prefixModeratorAddress}
	KeyExchanges           = []byte{prefixExchanges}
	KeyAdmins              = []byte{prefixAdmins}
	KeyNextExchangeId      = []byte{prefixNextExchangeId}
	KeyRatemeter           = []byte{prefixRatemeter}
	keyAddressRequestCount = []byte{prefixAddressRequestCount}
)

// default keys
var (
	KeyExchangeId             = "id"
	KeyExchangeReserveAddress = "reserve_address"
	KeyExchangeAdminAddress   = "admin_address"
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
	KeyExchangeAToBLimit      = "a_to_b_limit"
	KeyExchangeBtoALimit      = "b_to_a_limit"
	KeyExchangeAccumulatedFee = "accumulated_fee"
	RequiredKeysExchange      = []*string{
		&KeyExchangeReserveAddress,
		&KeyExchangeAdminAddress,
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
		&KeyExchangeAToBLimit,
		&KeyExchangeBtoALimit,
	}
)

func GetAddressRequestCountPrefix(address string) []byte {
	return append(keyAddressRequestCount, []byte(address)...)
}
