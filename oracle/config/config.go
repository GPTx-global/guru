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
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/viper"
)

var (
	once   sync.Once
	Config *configure
)

var (
	DaemonDir      = flag.String("daemon-dir", defaultDir(), "daemon directory")
	keyringBackend = flag.String("keyring-backend", "test", "keyring backend")
	rpcEndpoint    = flag.String("rpc-endpoint", "http://localhost:26657", "RPC endpoint for connecting to the blockchain node")
	chainID        = flag.String("chain-id", "guru_3110-1", "Chain ID for the blockchain network")
	keyName        = flag.String("key-name", "node1", "Key name for signing transactions")
	gasPrice       = flag.String("gas-price", "630000000000aguru", "Gas price for transactions (format: amount + denomination)")
	gasLimit       = flag.Uint64("gas-limit", 30000, "Gas limit for transactions")
)

type configure struct {
	rpcEndpoint string
	chainID     string
	keyringDir  string
	keyName     string
	gasPrice    string
	gasLimit    uint64

	address string
}

// LoadConfig load config from config.toml file
func LoadConfig() error {
	var err error
	once.Do(func() {
		flag.Parse()
		err = loadOrGenerateConfig()
	})

	return err
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

// SetAddress sets the address for the configured key
func (c *configure) SetAddress(address string) {
	c.address = address
}

// Address returns the address for the configured key
func (c *configure) Address() string {
	return c.address
}

// Keyring creates and returns a keyring instance for cryptographic operations
func (c *configure) Keyring() (keyring.Keyring, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	var backend string
	switch *keyringBackend {
	case "test":
		backend = keyring.BackendTest
	case "file":
		backend = keyring.BackendFile
	case "os":
		backend = keyring.BackendOS
	default:
		return nil, fmt.Errorf("invalid keyring backend: %s", *keyringBackend)
	}

	kr, err := keyring.New("guru", backend, c.keyringDir, nil, encCfg.Codec, hd.EthSecp256k1Option())
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return kr, nil
}

// PrintConfigInfo prints the current configuration information for debugging
func PrintConfigInfo() {
	if Config == nil {
		fmt.Println("[CONFIG] Configuration not loaded")
		return
	}

	fmt.Println("=== Oracle Daemon Configuration ===")
	fmt.Printf("Chain ID: %s\n", Config.chainID)
	fmt.Printf("RPC Endpoint: %s\n", Config.rpcEndpoint)
	fmt.Printf("Key Name: %s\n", Config.keyName)
	fmt.Printf("Keyring Backend: %s\n", *keyringBackend)
	fmt.Printf("Keyring Directory: %s\n", Config.keyringDir)
	fmt.Printf("Gas Price: %s\n", Config.gasPrice)
	fmt.Printf("Gas Limit: %d\n", Config.gasLimit)
	fmt.Println("================================")
}

func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(home, ".oracled")
}

func loadOrGenerateConfig() error {
	// 데몬 디렉토리가 존재하지 않으면 생성
	if err := os.MkdirAll(*DaemonDir, 0755); err != nil {
		return fmt.Errorf("failed to create daemon directory %s: %w", *DaemonDir, err)
	}

	daemonCfg := filepath.Join(*DaemonDir, "config.toml")

	// 설정 파일이 존재하지 않으면 기본 설정 파일 생성
	if _, err := os.Stat(daemonCfg); os.IsNotExist(err) {
		fmt.Printf("[CONFIG] Config file not found at %s, creating default config file\n", daemonCfg)

		if err := generateDefaultConfigFile(daemonCfg); err != nil {
			return fmt.Errorf("failed to generate default config file: %w", err)
		}

		fmt.Printf("[CONFIG] Default config file created at %s\n", daemonCfg)
	}

	// 생성된 또는 기존 설정 파일 로드
	if err := loadConfigFromFile(daemonCfg); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", daemonCfg, err)
	}

	fmt.Printf("[CONFIG] Successfully loaded config from %s\n", daemonCfg)
	return nil
}

// generateDefaultConfigFile creates a default config.toml file with proper values
func generateDefaultConfigFile(configPath string) error {
	if err := os.MkdirAll(*DaemonDir, 0755); err != nil {
		return fmt.Errorf("failed to create keyring directory %s: %w", *DaemonDir, err)
	}

	defaultConfigContent := fmt.Sprintf(`# Oracle Daemon Configuration File
# Generated automatically on startup

# RPC endpoint for connecting to the blockchain node  
rpc_endpoint = "%s"

# Chain ID for the blockchain network
chain_id = "%s"

# Directory path for keyring storage (absolute path recommended)
keyring_dir = "%s"

# Key name for signing transactions
key_name = "%s"

# Gas price for transactions (format: amount + denomination)
gas_price = "%s"

# Gas limit for transactions
gas_limit = %d
`, *rpcEndpoint, *chainID, *DaemonDir, *keyName, *gasPrice, *gasLimit)

	if err := os.WriteFile(configPath, []byte(defaultConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadConfigFromFile load config from config.toml file
func loadConfigFromFile(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
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

	return nil
}

// ValidateConfig validates the current configuration values
func ValidateConfig() error {
	if Config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// RPC endpoint 검증
	if Config.rpcEndpoint == "" {
		return fmt.Errorf("rpc_endpoint cannot be empty")
	}

	// Chain ID 검증
	if Config.chainID == "" {
		return fmt.Errorf("chain_id cannot be empty")
	}

	// Keyring 디렉토리 검증
	if Config.keyringDir == "" {
		return fmt.Errorf("keyring_dir cannot be empty")
	}

	// Keyring 디렉토리 존재 여부 확인 및 생성
	// if err := os.MkdirAll(Config.keyringDir, 0755); err != nil {
	// 	return fmt.Errorf("failed to create keyring directory %s: %w", Config.keyringDir, err)
	// }

	// Key name 검증
	if Config.keyName == "" {
		return fmt.Errorf("key_name cannot be empty")
	}

	// Gas price 검증
	if Config.gasPrice == "" {
		return fmt.Errorf("gas_price cannot be empty")
	}

	// Gas limit 검증
	if Config.gasLimit == 0 {
		return fmt.Errorf("gas_limit must be greater than 0")
	}

	return nil
}
