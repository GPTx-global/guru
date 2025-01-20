package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/GPTx-global/guru/tests/e2e/upgrade"
)

const (
	// defaultManagerNetwork defines the network used by the upgrade manager
	defaultManagerNetwork = "guru-local"

	// blocksAfterUpgrade defines how many blocks must be produced after an upgrade is
	// considered successful
	blocksAfterUpgrade = 5

	// relatedBuildPath defines the path where the build data is stored
	relatedBuildPath = "../../build/"

	// registryDockerFile builds the image using the docker image registry
	registryDockerFile = "./upgrade/Dockerfile.init"

	// repoDockerFile builds the image from the repository (used when the images are not pushed to the registry, e.g. main)
	repoDockerFile = "./Dockerfile.repo"
)

type IntegrationTestSuite struct {
	suite.Suite

	upgradeManager *upgrade.Manager
	upgradeParams  upgrade.Params
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")
}

// TearDownSuite kills the running container, removes the network and mount path
func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down e2e integration test suite...")
}
