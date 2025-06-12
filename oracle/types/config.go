package types

import (
	"fmt"
	"os"
	"sync"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/crypto/hd"
	"github.com/GPTx-global/guru/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/viper"
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

// LoadConfig load config from config.toml file
func LoadConfig(configPath ...string) error {
	var err error
	once.Do(func() {
		err = loadConfigFromFile(configPath...)
	})
	return err
}

// loadConfigFromFile load config from config.toml file
func loadConfigFromFile(configPath ...string) error {
	var configFile string

	if len(configPath) > 0 && configPath[0] != "" {
		configFile = configPath[0]
	} else {
		defaultPaths := []string{
			"/Users/pamla/.oracled/config.toml",
			"./config.toml",
			"../config/config.toml",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				configFile = path
				break
			}
		}

		if configFile == "" {
			fmt.Println("[CONFIG] config.toml not found, using default values")
			Config = &configure{
				rpcEndpoint: "http://localhost:26657",
				chainID:     "guru_3110-1",
				keyringDir:  "../",
				keyName:     "node1",
				gasPrice:    "630000000aguru",
				gasLimit:    30000,
			}
			return nil
		}
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	Config = &configure{
		rpcEndpoint: v.GetString("rpc_endpoint"),
		chainID:     v.GetString("chain_id"),
		keyringDir:  v.GetString("keyring_dir"),
		keyName:     v.GetString("key_name"),
		gasPrice:    v.GetString("gas_price"),
		gasLimit:    v.GetUint64("gas_limit"),
	}

	if Config.rpcEndpoint == "" {
		Config.rpcEndpoint = "http://localhost:26657"
	}
	if Config.chainID == "" {
		Config.chainID = "guru_3110-1"
	}
	if Config.keyringDir == "" {
		Config.keyringDir = "../"
	}
	if Config.keyName == "" {
		Config.keyName = "node1"
	}
	if Config.gasPrice == "" {
		Config.gasPrice = "630000000aguru"
	}
	if Config.gasLimit == 0 {
		Config.gasLimit = 30000
	}

	return nil
}

// RpcEndpoint returns the RPC endpoint URL
func (c *configure) RpcEndpoint() string {
	return c.rpcEndpoint
}

// ChainID returns the blockchain chain ID
func (c *configure) ChainID() string {
	return c.chainID
}

// KeyringDir returns the keyring directory path
func (c *configure) KeyringDir() string {
	return c.keyringDir
}

// KeyName returns the key name for signing transactions
func (c *configure) KeyName() string {
	return c.keyName
}

// GasPrice returns the gas price for transactions
func (c *configure) GasPrice() string {
	return c.gasPrice
}

// GasLimit returns the gas limit for transactions
func (c *configure) GasLimit() uint64 {
	return c.gasLimit
}

// Keyring creates and returns a keyring instance for cryptographic operations
func (c *configure) Keyring() (keyring.Keyring, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	kr, err := keyring.New("guru", keyring.BackendTest, c.keyringDir, nil, encCfg.Codec, hd.EthSecp256k1Option())
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return kr, nil
}
