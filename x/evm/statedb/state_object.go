// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE
package statedb

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"github.com/GPTx-global/guru/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Code []byte

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the storage of auth module.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	CodeHash []byte
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *types.StateAccount {
	return &types.StateAccount{
		Balance:  new(big.Int),
		CodeHash: types.EmptyCodeHash.Bytes(),
		Root:     types.EmptyRootHash,
	}
}

// IsContract returns if the account contains contract code.
func (acct Account) IsContract() bool {
	return !bytes.Equal(acct.CodeHash, types.EmptyCodeHash.Bytes())
}

// Storage represents in-memory cache/buffer of contract storage.
type Storage map[common.Hash]common.Hash

// SortedKeys sort the keys for deterministic iteration
func (s Storage) SortedKeys() []common.Hash {
	keys := make([]common.Hash, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) < 0
	})
	return keys
}

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage, len(s))
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// stateObject is the state of an acount
type stateObject struct {
	db       *StateDB
	address  common.Address      // address of ethereum account
	addrHash common.Hash         // hash of ethereum address of the account
	origin   *types.StateAccount // Account original data without any change applied, nil means it was not existent
	data     types.StateAccount  // Account data with all mutations applied in the scope of block

	// Write caches.
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution, reset for every transaction

	// Cache flags.
	dirtyCode bool // true if the code was updated

	// Flag whether the account was marked as self-destructed. The self-destructed account
	// is still accessible in the scope of same transaction.
	selfDestructed bool

	// Flag whether the account was marked as deleted. A self-destructed account
	// or an account that is considered as empty will be marked as deleted at
	// the end of transaction and no longer accessible anymore.
	deleted bool

	// Flag whether the object was created in the current transaction
	created bool
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, data types.StateAccount) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = types.EmptyCodeHash.Bytes()
	}
	if data.Root == (common.Hash{}) {
		data.Root = emptyRoot
	}
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		data:           data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, types.EmptyCodeHash.Bytes())
}

func (s *stateObject) markSelfdestructed() {
	s.selfDestructed = true
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateObject) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.Balance(), amount))
}

// SetBalance update account balance.
func (s *stateObject) SetBalance(amount *big.Int) {
	s.db.journal.append(balanceChange{
		account: &s.address,
		prev:    new(big.Int).Set(s.data.Balance),
	})
	s.setBalance(amount)
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.data.Balance = amount
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash.Bytes()) {
		return nil
	}
	code := s.db.keeper.GetCode(s.db.ctx, common.BytesToHash(s.CodeHash()))
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none.
func (s *stateObject) CodeSize() int {
	return len(s.Code())
}

// SetCode set contract code to account
func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code()
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

// SetCode set nonce to account
func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

// CodeHash returns the code hash of account
func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

// Balance returns the balance of account
func (s *stateObject) Balance() *big.Int {
	return s.data.Balance
}

// Nonce returns the nonce of account
func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

// GetCommittedState query the committed state
func (s *stateObject) GetCommittedState(key common.Hash) common.Hash {
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If no live objects are available, load it from keeper
	value := s.db.keeper.GetState(s.db.ctx, s.Address(), key)
	s.originStorage[key] = value
	return value
}

// GetState query the current state (including dirty state)
func (s *stateObject) GetState(key common.Hash) common.Hash {
	if value, dirty := s.dirtyStorage[key]; dirty {
		return value
	}
	return s.GetCommittedState(key)
}

// SetState sets the contract state
func (s *stateObject) SetState(key common.Hash, value common.Hash) {
	// If the new value is the same as old, don't set
	prev := s.GetState(key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}
