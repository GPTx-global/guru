package e2e

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/GPTx-global/guru/tests/e2e/docker"
)

const (
	// defaultManagerNetwork defines the network used by the upgrade manager
	defaultManagerNetwork = "guru-local"

	// relatedBuildPath defines the path where the build data is stored
	relatedBuildPath = "../../build/"

	// repoDockerFile builds the image from the repository (used when the images are not pushed to the registry, e.g. main)
	repoDockerFile = "./docker/Dockerfile.repo"

	// Chain ID
	chainID = "guru_3110-1"
)

type IntegrationTestSuite struct {
	suite.Suite

	manager *docker.Manager
	// upgradeParams  upgrade.Params
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")
	var err error

	s.manager, err = docker.NewManager(defaultManagerNetwork)
	s.Require().NoError(err, "upgrade manager creation error")
	if _, err := os.Stat(relatedBuildPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(relatedBuildPath, os.ModePerm)
		s.Require().NoError(err, "can't create build tmp dir")
	}
}

// TearDownSuite kills the running container
func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down e2e integration test suite...")
	s.T().Log("killing nodes...")
	err := s.manager.KillAllNodes()
	s.Require().NoError(err, "can't kill nodes")
	s.T().Log("removing nodes...")
	err = s.manager.RemoveAllNodes()
	s.Require().NoError(err, "can't remove nodes")
	s.T().Log("removing network...")
	s.Require().NoError(s.manager.RemoveNetwork(), "can't remove network")
}

// runNodeWithBranch builds a docker image using the given branch of the Guru repository.
// Before running the node, runs a script to modify some configurations for the tests
// (e.g.: gov proposal voting period, setup accounts, balances, etc..)
// After a successful build, runs the container.
func (s *IntegrationTestSuite) runNodeWithBranch(branch string) {
	s.T().Log("running a node with branch", branch)

	// check if the node is running this the same branch
	if s.manager.Nodes[branch] != nil {
		s.T().Log("killing the existing node with branch", branch)
		s.manager.KillNode(branch)
		s.manager.RemoveNode(branch)
	}

	var (
		name    = "e2e-test/guru"
		version = branch
	)
	var err error

	err = s.manager.BuildImage(
		name,
		version,
		repoDockerFile,
		".",
		map[string]string{"BRANCH_NAME": branch},
	)
	s.Require().NoError(err, "can't build container for e2e test")
	node := docker.NewNode(name, version)
	node.SetEnvVars([]string{fmt.Sprintf("CHAIN_ID=%s", chainID)})
	// node.SetEnvVars([]string{fmt.Sprintf("CHAIN_ID=%s", s.upgradeParams.ChainID)})

	err = s.manager.RunNode(node)
	s.Require().NoError(err, "can't run node Guru using branch %s", branch)
}

func (s *IntegrationTestSuite) runNodeIfNotExist(branch string) {
	if s.manager.Nodes[branch] == nil {
		s.runNodeWithBranch(branch)
	}
}
