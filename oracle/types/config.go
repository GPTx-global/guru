package types

import (
	"fmt"
	"sync"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/crypto/hd"
	"github.com/GPTx-global/guru/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

var (
	once   sync.Once
	Config *configure
)

type configure struct {
	rpcEndpoint string
	chainID     string
	keyringDir  string
	keyName     string
	gasPrice    string
	gasLimit    uint64
}

// TODO: load config from file
func LoadConfig() {
	once.Do(func() {
		Config = new(configure)
		Config.rpcEndpoint = "http://localhost:26657"
		Config.chainID = "guru_3110-1"
		Config.keyringDir = "../"
		Config.keyName = "node1"
		Config.gasPrice = "630000000aguru"
		Config.gasLimit = 30000
	})
}

func (c *configure) RpcEndpoint() string {
	return c.rpcEndpoint
}

func (c *configure) ChainID() string {
	return c.chainID
}

func (c *configure) KeyringDir() string {
	return c.keyringDir
}

func (c *configure) KeyName() string {
	return c.keyName
}

func (c *configure) GasPrice() string {
	return c.gasPrice
}

func (c *configure) GasLimit() uint64 {
	return c.gasLimit
}

func (c *configure) Keyring() (keyring.Keyring, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	kr, err := keyring.New("guru", keyring.BackendTest, c.keyringDir, nil, encCfg.Codec, hd.EthSecp256k1Option())
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return kr, nil
}
