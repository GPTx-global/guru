package client

import (
	"os"
	"path/filepath"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

type Config struct {
	rpcEndpoint  string
	chainID      string
	keyringDir   string
	keyName      string
	gasPrice     string
	gasLimit     uint64
	oracleEvents []string
}

func LoadConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		rpcEndpoint: "http://localhost:26657",
		chainID:     "guru_3110-1",
		keyringDir:  filepath.Join(homeDir, ".guru"),
		keyName:     "oracle",
		gasPrice:    "20000000000aguru", // 20 gwei
		gasLimit:    200000,
		oracleEvents: []string{
			"tm.event='Tx' AND oracle_request.id EXISTS",
		},
	}
}

func (c *Config) GetKeyring() (keyring.Keyring, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	return keyring.New("guru", keyring.BackendFile, c.keyringDir, nil, encCfg.Codec)
}
