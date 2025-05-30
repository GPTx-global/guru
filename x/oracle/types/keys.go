package types

import (
	"encoding/binary"
	"fmt"
)

const (
	// ModuleName defines the module name
	ModuleName = "oracle"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_oracle"
)

// KV Store key prefix bytes
const (
	preficParams = iota + 1
	prefixModeratorAddress
	prefixOracleRequestDoc
	prefixOracleRequestDocCount
	prefixOracleData
)

// KV Store key prefixes
var (
	KeyParams                = []byte{preficParams}
	KeyModeratorAddress      = []byte{prefixModeratorAddress}
	KeyOracleRequestDoc      = []byte{prefixOracleRequestDoc}
	KeyOracleRequestDocCount = []byte{prefixOracleRequestDocCount}
	KeyOracleData            = []byte{prefixOracleData}
)

func IDToBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetOracleRequestDocKey returns the key for storing OracleRequsetDoc
func GetOracleRequestDocKey(id uint64) []byte {
	return append(KeyOracleRequestDoc, IDToBytes(id)...)
}

// GetOracleDataKey returns the key for storing oracle data
func GetOracleDataKey(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return append(KeyOracleData, bz...)
}

// ParseOracleDataKey parses the oracle data key and returns the ID
func ParseOracleDataKey(key []byte) (uint64, error) {
	if len(key) != 9 {
		return 0, fmt.Errorf("invalid oracle data key length: %d", len(key))
	}
	return binary.BigEndian.Uint64(key[1:]), nil
}
