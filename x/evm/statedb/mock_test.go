package statedb_test

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/GPTx-global/guru/x/evm/statedb"
	"github.com/GPTx-global/guru/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	_             statedb.Keeper = &MockKeeper{}
	errAddress    common.Address = common.BigToAddress(big.NewInt(100))
	emptyCodeHash                = crypto.Keccak256(nil)
)

type MockAcount struct {
	account types.StateAccount
	states  statedb.Storage
}

type MockKeeper struct {
	accounts map[common.Address]MockAcount
	codes    map[common.Hash][]byte
}

func NewMockKeeper() *MockKeeper {
	return &MockKeeper{
		accounts: make(map[common.Address]MockAcount),
		codes:    make(map[common.Hash][]byte),
	}
}

func (k MockKeeper) GetAccount(_ sdk.Context, addr common.Address) *types.StateAccount {
	acct, ok := k.accounts[addr]
	if !ok {
		return nil
	}
	return &acct.account
}

func (k MockKeeper) GetState(_ sdk.Context, addr common.Address, key common.Hash) common.Hash {
	return k.accounts[addr].states[key]
}

func (k MockKeeper) GetCode(_ sdk.Context, codeHash common.Hash) []byte {
	return k.codes[codeHash]
}

func (k MockKeeper) ForEachStorage(_ sdk.Context, addr common.Address, cb func(key, value common.Hash) bool) {
	if acct, ok := k.accounts[addr]; ok {
		for k, v := range acct.states {
			if !cb(k, v) {
				return
			}
		}
	}
}

func (k MockKeeper) SetAccount(_ sdk.Context, addr common.Address, account types.StateAccount) error {
	if addr == errAddress {
		return errors.New("mock db error")
	}
	acct, exists := k.accounts[addr]
	if exists {
		// update
		acct.account = account
		k.accounts[addr] = acct
	} else {
		k.accounts[addr] = MockAcount{account: account, states: make(statedb.Storage)}
	}
	return nil
}

func (k MockKeeper) SetState(_ sdk.Context, addr common.Address, key common.Hash, value []byte) {
	if acct, ok := k.accounts[addr]; ok {
		if len(value) == 0 {
			delete(acct.states, key)
		} else {
			acct.states[key] = common.BytesToHash(value)
		}
	}
}

func (k MockKeeper) SetCode(_ sdk.Context, codeHash []byte, code []byte) {
	k.codes[common.BytesToHash(codeHash)] = code
}

func (k MockKeeper) DeleteAccount(_ sdk.Context, addr common.Address) error {
	if addr == errAddress {
		return errors.New("mock db error")
	}
	old := k.accounts[addr]
	delete(k.accounts, addr)
	if !bytes.Equal(old.account.CodeHash, emptyCodeHash) {
		delete(k.codes, common.BytesToHash(old.account.CodeHash))
	}
	return nil
}

func (k MockKeeper) Clone() *MockKeeper {
	accounts := make(map[common.Address]MockAcount, len(k.accounts))
	for k, v := range k.accounts {
		accounts[k] = v
	}
	codes := make(map[common.Hash][]byte, len(k.codes))
	for k, v := range k.codes {
		codes[k] = v
	}
	return &MockKeeper{accounts, codes}
}
