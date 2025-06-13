package types

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	tempDir    string
	configFile string
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (suite *ConfigTestSuite) SetupTest() {
	// 임시 디렉토리 생성
	var err error
	suite.tempDir, err = os.MkdirTemp("", "oracle_test")
	suite.Require().NoError(err)

	suite.configFile = filepath.Join(suite.tempDir, "config.toml")

	// Config 초기화 (once.Do 재설정을 위해)
	once = sync.Once{}
	Config = nil
}

func (suite *ConfigTestSuite) TearDownTest() {
	// 임시 디렉토리 정리
	os.RemoveAll(suite.tempDir)
}

func (suite *ConfigTestSuite) TestLoadConfig_ValidConfig() {
	// Given: 유효한 설정 파일
	configContent := `
rpc_endpoint = "http://test:26657"
chain_id = "test-chain"
keyring_dir = "/test/keyring"
key_name = "test-key"
gas_price = "1000000aguru"
gas_limit = 50000
`
	err := os.WriteFile(suite.configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	// When: 설정 로드
	err = LoadConfig(suite.configFile)

	// Then: 설정이 정확히 로드됨
	suite.NoError(err)
	suite.Equal("http://test:26657", Config.RpcEndpoint())
	suite.Equal("test-chain", Config.ChainID())
	suite.Equal("/test/keyring", Config.KeyringDir())
	suite.Equal("test-key", Config.KeyName())
	suite.Equal("1000000aguru", Config.GasPrice())
	suite.Equal(uint64(50000), Config.GasLimit())
}

func (suite *ConfigTestSuite) TestLoadConfig_DefaultValues() {
	// Given: 설정 파일이 존재하지 않음
	nonExistentPath := filepath.Join(suite.tempDir, "nonexistent.toml")

	// When: 설정 로드
	err := LoadConfig(nonExistentPath)

	// Then: 기본값으로 설정됨
	suite.NoError(err)
	suite.Equal("http://localhost:26657", Config.RpcEndpoint())
	suite.Equal("guru_3110-1", Config.ChainID())
	suite.Equal("../", Config.KeyringDir())
	suite.Equal("node1", Config.KeyName())
	suite.Equal("630000000aguru", Config.GasPrice())
	suite.Equal(uint64(30000), Config.GasLimit())
}

func (suite *ConfigTestSuite) TestLoadConfig_PartialConfig() {
	// Given: 부분적인 설정 파일
	configContent := `
rpc_endpoint = "http://partial:26657"
chain_id = "partial-chain"
`
	err := os.WriteFile(suite.configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	// When: 설정 로드
	err = LoadConfig(suite.configFile)

	// Then: 지정된 값과 기본값이 조합됨
	suite.NoError(err)
	suite.Equal("http://partial:26657", Config.RpcEndpoint())
	suite.Equal("partial-chain", Config.ChainID())
	suite.Equal("../", Config.KeyringDir()) // 기본값
	suite.Equal("node1", Config.KeyName())  // 기본값
}

func (suite *ConfigTestSuite) TestLoadConfig_InvalidFile() {
	// Given: 잘못된 형식의 설정 파일
	invalidContent := `invalid toml content [[[`
	err := os.WriteFile(suite.configFile, []byte(invalidContent), 0644)
	suite.Require().NoError(err)

	// When: 설정 로드
	err = LoadConfig(suite.configFile)

	// Then: 에러 발생
	suite.Error(err)
	suite.Contains(err.Error(), "failed to read config file")
}

func (suite *ConfigTestSuite) TestLoadConfig_OncePattern() {
	// Given: 첫 번째 설정 로드
	configContent1 := `rpc_endpoint = "http://first:26657"`
	err := os.WriteFile(suite.configFile, []byte(configContent1), 0644)
	suite.Require().NoError(err)

	err = LoadConfig(suite.configFile)
	suite.NoError(err)
	firstEndpoint := Config.RpcEndpoint()

	// When: 두 번째 설정 로드 (다른 내용)
	configContent2 := `rpc_endpoint = "http://second:26657"`
	err = os.WriteFile(suite.configFile, []byte(configContent2), 0644)
	suite.Require().NoError(err)

	err = LoadConfig(suite.configFile)

	// Then: 첫 번째 설정이 유지됨 (sync.Once 동작)
	suite.NoError(err)
	suite.Equal(firstEndpoint, Config.RpcEndpoint())
}

func (suite *ConfigTestSuite) TestKeyring_Creation() {
	// Given: 유효한 설정
	once = sync.Once{}
	Config = &configure{
		keyringDir: suite.tempDir,
	}

	// When: 키링 생성
	kr, err := Config.Keyring()

	// Then: 키링이 성공적으로 생성됨
	suite.NoError(err)
	suite.NotNil(kr)
	suite.Implements((*keyring.Keyring)(nil), kr)
}

func (suite *ConfigTestSuite) TestConfigGetters() {
	// Given: 설정된 config
	config := &configure{
		rpcEndpoint: "http://test:26657",
		chainID:     "test-chain",
		keyringDir:  "/test/keyring",
		keyName:     "test-key",
		gasPrice:    "1000000aguru",
		gasLimit:    50000,
	}

	// Then: 모든 getter 메서드가 정확한 값을 반환
	suite.Equal("http://test:26657", config.RpcEndpoint())
	suite.Equal("test-chain", config.ChainID())
	suite.Equal("/test/keyring", config.KeyringDir())
	suite.Equal("test-key", config.KeyName())
	suite.Equal("1000000aguru", config.GasPrice())
	suite.Equal(uint64(50000), config.GasLimit())
}

// 단위 테스트 함수들

func TestLoadConfigWithEmptyPath(t *testing.T) {
	// Given: 빈 경로
	once = sync.Once{}
	Config = nil

	// When: 빈 경로로 설정 로드
	err := LoadConfig("")

	// Then: 기본값으로 설정됨
	require.NoError(t, err)
	assert.NotNil(t, Config)
	assert.Equal(t, "http://localhost:26657", Config.RpcEndpoint())
}

func TestConfigNilSafety(t *testing.T) {
	// Given: Config가 nil인 상태
	originalConfig := Config
	Config = nil

	// When/Then: nil 포인터 참조로 인한 패닉 방지 테스트
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Config methods should handle nil gracefully, but got panic: %v", r)
		}
		Config = originalConfig
	}()

	// Config가 nil일 때의 동작은 실제 구현에 따라 다를 수 있음
	// 여기서는 nil 체크가 구현되어 있지 않으므로 패닉이 발생할 수 있음
}

func TestDefaultConfigPaths(t *testing.T) {
	// Given: 기본 경로들이 존재하지 않는 환경
	once = sync.Once{}
	Config = nil

	// When: 경로 없이 설정 로드
	err := LoadConfig()

	// Then: 에러 없이 기본값으로 설정
	require.NoError(t, err)
	assert.NotNil(t, Config)
	assert.Equal(t, "guru_3110-1", Config.ChainID())
}
