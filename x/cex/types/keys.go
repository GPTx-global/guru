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
	prefixReserveAccount
	prefixRate
	prefixAdmin
	prefixIbcDenom
)

// KV Store key prefixes
var (
	KeyModeratorAddress = []byte{prefixModeratorAddress}
	KeyReserveAccount   = []byte{prefixReserveAccount}
	KeyRate             = []byte{prefixRate}
	KeyAdmin            = []byte{prefixAdmin}
	KeyIbcDenom         = []byte{prefixIbcDenom}
)

// func GetCoinPairKeyKey(pairDenom string) []byte {
// 	return append(KeyCoinPair, []byte(pairDenom)...)
// }
