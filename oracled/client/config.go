package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/crypto/hd"
	"github.com/GPTx-global/guru/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

type Config struct {
	rpcEndpoint string
	chainID     string
	keyringDir  string
	keyName     string
	gasPrice    string
	gasLimit    uint64
}

func LoadConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		// rpcEndpoint: "http://192.168.48.131:26657",
		rpcEndpoint: "http://localhost:26657",
		chainID:     "guru_3110-1",
		keyringDir:  filepath.Join(homeDir, ".gurud"),
		keyName:     "node0",
		gasPrice:    "20000000000aguru", // 20 gwei
		gasLimit:    200000,
	}
}

func (c *Config) GetKeyring() (keyring.Keyring, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	// 테스트 환경용
	kr, err := keyring.New(
		"guru",                  // service name
		keyring.BackendTest,     // test backend
		c.keyringDir,            // home directory
		nil,                     // input reader
		encCfg.Codec,            // codec
		hd.EthSecp256k1Option(), // 개발용 옵션
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return kr, nil
}
