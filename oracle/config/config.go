package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/crypto/hd"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pelletier/go-toml/v2"
)

var home = flag.String("home", homeDir(), "oracle daemon home directory")

var (
	globalConfig configData
	mu           sync.Mutex
)

type configData struct {
	Chain chainConfig `toml:"chain"`
	Key   keyConfig   `toml:"key"`
	Gas   gasConfig   `toml:"gas"`
}

type chainConfig struct {
	ID       string `toml:"id"`
	Endpoint string `toml:"endpoint"`
}

type keyConfig struct {
	Name           string `toml:"name"`
	KeyringDir     string `toml:"keyring_dir"`
	KeyringBackend string `toml:"keyring_backend"`
}

type gasConfig struct {
	Limit  uint64 `toml:"limit"`
	Prices string `toml:"prices"`
}

func Load() {
	flag.Parse()
	path := filepath.Join(Home(), "config.toml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefaultConfig(path); err != nil {
			log.Fatalf("Failed to create default config: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	if err := toml.Unmarshal(data, &globalConfig); err != nil {
		log.Fatalf("Failed to parse TOML: %v", err)
	}

	if err := validateConfig(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	log.Infof("Loaded config from %s", path)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	return filepath.Join(home, ".oracled")
}

func createDefaultConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	globalConfig = configData{
		Chain: chainConfig{
			ID:       "guru_3110-1",
			Endpoint: "http://localhost:26657",
		},
		Key: keyConfig{
			Name:           "node1",
			KeyringDir:     Home(),
			KeyringBackend: "test",
		},
		Gas: gasConfig{
			Limit:  30000,
			Prices: "630000000000aguru",
		},
	}

	data, err := toml.Marshal(globalConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal TOML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func validateConfig() error {
	if globalConfig.Chain.ID == "" {
		return fmt.Errorf("chain ID is required")
	}

	if globalConfig.Chain.Endpoint == "" {
		return fmt.Errorf("chain endpoint is required")
	}

	if globalConfig.Key.Name == "" {
		return fmt.Errorf("key name is required")
	}

	if globalConfig.Key.KeyringDir == "" {
		return fmt.Errorf("keyring directory is required")
	}

	if globalConfig.Key.KeyringBackend == "" {
		return fmt.Errorf("keyring backend is required")
	}

	if globalConfig.Gas.Limit == 0 {
		return fmt.Errorf("gas limit is required")
	}

	if globalConfig.Gas.Prices == "" {
		return fmt.Errorf("gas prices is required")
	}

	return nil
}

func Print() {
	log.Infof("%-15s: %s", "Home", Home())
	log.Infof("%-15s: %s", "Chain ID", ChainID())
	log.Infof("%-15s: %s", "Chain Endpoint", ChainEndpoint())
	log.Infof("%-15s: %s", "Key Name", KeyName())
	log.Infof("%-15s: %s", "Keyring Dir", KeyringDir())
	log.Infof("%-15s: %s", "Keyring Backend", KeyringBackend())
	log.Infof("%-15s: %s", "Address", Address().String())
	log.Infof("%-15s: %d", "Gas Limit", GasLimit())
	log.Infof("%-15s: %s", "Gas Prices", GasPrices())
}

func Home() string {
	return *home
}

func ChainID() string {
	return globalConfig.Chain.ID
}

func ChainEndpoint() string {
	return globalConfig.Chain.Endpoint
}

func KeyName() string {
	return globalConfig.Key.Name
}

func KeyringDir() string {
	return globalConfig.Key.KeyringDir
}

func KeyringBackend() string {
	return globalConfig.Key.KeyringBackend
}

func Keyring() keyring.Keyring {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	var backend string
	switch KeyringBackend() {
	case "test":
		backend = keyring.BackendTest
	case "file":
		backend = keyring.BackendFile
	case "os":
		backend = keyring.BackendOS
	default:
		log.Fatalf("Invalid keyring backend: %s", KeyringBackend())
	}

	kr, err := keyring.New("guru", backend, KeyringDir(), nil, encCfg.Codec, hd.EthSecp256k1Option())
	if err != nil {
		log.Fatalf("Failed to create keyring: %v", err)
	}

	return kr
}

func Address() sdk.AccAddress {
	kr := Keyring()
	info, err := kr.Key(KeyName())
	if err != nil {
		log.Fatalf("Failed to get key info: %v", err)
	}

	address, err := info.GetAddress()
	if err != nil {
		log.Fatalf("Failed to get address: %v", err)
	}

	return address
}

func GasLimit() uint64 {
	return globalConfig.Gas.Limit
}

func GasPrices() string {
	mu.Lock()
	defer mu.Unlock()

	return globalConfig.Gas.Prices
}

func SetGasPrice(gasPrice string) {
	mu.Lock()
	defer mu.Unlock()

	globalConfig.Gas.Prices = gasPrice
	// log.Debugf("gas price updated: %s", gasPrice)
}

func ChannelSize() int {
	return 1 << 10
}

func SetForTesting(id, endpoint, keyName, keyringDir, keyringBackend, gasPrices string, gasLimit uint64) {
	globalConfig = configData{
		Chain: chainConfig{
			ID:       id,
			Endpoint: endpoint,
		},
		Key: keyConfig{
			Name:           keyName,
			KeyringDir:     keyringDir,
			KeyringBackend: keyringBackend,
		},
		Gas: gasConfig{
			Limit:  gasLimit,
			Prices: gasPrices,
		},
	}
}
