package itest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"

	"github.com/GPTx-global/guru/tests/itest/docker"
	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
)

const (
	// relatedBuildPath defines the path where the build data is stored
	relatedBuildPath = "../../build/"

	// repoDockerFile builds the image from the repository (used when the images are not pushed to the registry, e.g. main)
	repoDockerFileVal  = "./docker/Dockerfile.val"
	repoDockerFileFull = "./docker/Dockerfile.full"
	// genesisFile        = "./docker/genesis.json"

	// Chain ID
	chainID = "guru_3110-1"
)

var accounts = []string{
	"guru10jmp6sgh4cc6zt3e8gw05wavvejgr5pwggsdaj", // base addr
	"guru1cml96vmptgw99syqrrz8az79xer2pcgpawsch9", // mod addr
	"guru1jcltmuhplrdcwp7stlr4hlhlhgd4htqhtx0smk", // tester1
	"guru1gzsvk8rruqn2sx64acfsskrwy8hvrmaf6dvhj3", // tester2
	"guru1fx944mzagwdhx0wz7k9tfztc8g3lkfk6ecee3f", // tester3
	// "guru174mfr668p349h6dxt52rfpsr0n85hux8w8m93r", // val 0: sender
	"guru1e2rnhff8v20n4v7xgt4sgrmwgl0ykqqw7ka9cf", // val 1
	"guru1y5v8wgnfhzx8xh4kq4knf0eat3qrw6kguclz03", // val 2
	"guru18c8ea0tuwh40dqecuvcdhlthetn8gexra94jg0", // val 3
	"guru15mu7d4th0acqdfulsgd4epsxcv02qlngg4sgvk", // val 4
	"guru17nywmzjr248agulkw584yt7qfvatq8dmp0ly06", // val 5
	"guru1796e7e8tyhxcfflh5f9qq2tf2d8n7ta5zfpf3v", // val 6
	"guru1c3du9k3xckzvehlakzk3rza0kqzakvkefp5kln", // val 7
	"guru1dp2nwka2s9vmlwafvh60qxk3xqmcs3gn27rvn5", // full 0
	"guru1gt2f3633jnwa58wstr49tk5c6f07r0ryx73x2q", // full 1
	"guru12wvafj5tw8kpl9cvcp63w9mjyql3ex6m66dzdk", // full 2
	"guru13nczxvpjl9n5saw9v3y6kzez0qzvq4ad4nj46t", // full 3
	"guru1ndm40uyhejegt3m9ts6q3znllwqd85egrlr2hw", // full 4
	"guru1a7drfnq02q6fej9a7tfrmnqp29jqpr37rsw92y", // full 5
	"guru10rucq3nqfjgwclfgz32y5ldsjwxtt0kt02j72v", // full 6
	"guru1qnsugp4ayulhrvx5lza5anw6p73kvxrut0uwxq", // full 7
}

type IntegrationTestSuite struct {
	suite.Suite

	manager *docker.Manager
}

var (
	repository         = "e2e-test/guru"
	repositoryFullNode = "e2e-test/guru-sentry"
	peers              map[string][]string
)

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")

	manager, err := docker.NewManager()
	s.Require().NoError(err)
	s.manager = manager
	peers = make(map[string][]string)

	if _, err := os.Stat(relatedBuildPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(relatedBuildPath, os.ModePerm)
		s.Require().NoError(err, "can't create build tmp dir")
	}
}

// TearDownSuite kills the running test suite
func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down e2e integration test suite...")
	s.T().Log("clearing the networks...")
	s.Require().NoError(s.manager.Clear(), "can't clear the networks")
}

// runNodeWithBranch builds a docker image using the given branch of the Guru repository.
// Before running the node, runs a script to modify some configurations for the tests
// (e.g.: gov proposal voting period, setup accounts, balances, etc..)
// After a successful build, runs the container.
func (s *IntegrationTestSuite) runNetworkWithBranch(ctx context.Context, node *helpers.NodeInfo) {
	branch := node.Branch
	nFulls := node.Fulls
	nVals := node.Vals

	s.T().Log("running a network with branch", branch)

	s.Require().LessOrEqual(nVals, 8, "maximum allowed number of validator nodes are 8")
	s.Require().LessOrEqual(nFulls, 8, "maximum allowed number of full nodes are 8")

	// Clear if the network exists
	if s.manager.Networks[branch] != nil {
		s.manager.ClearNetwork(branch)
	}

	// create new network
	err := s.manager.CreateNetwork(branch)
	s.Require().NoError(err)

	// build the image from given branch
	err = s.manager.BuildImage(
		repository,
		branch,
		repoDockerFileVal,
		".",
		map[string]string{
			"BRANCH_NAME": branch,
		},
	)
	s.Require().NoError(err, "can't build container for e2e test")

	// run the first validator node
	cmd := append([]string{"bash", "./init-validator-node.sh"}, node.GenesisParams...)
	s.T().Log("running validator 0")
	options := &dockertest.RunOptions{
		Name:       "validator_node_0",
		Repository: repository,
		Tag:        branch,
		Networks:   []*dockertest.Network{s.manager.Networks[branch]},
		Cmd:        cmd,
	}
	peer := s.runNewNode(ctx, options, branch, true, 0)
	peers[branch] = append(peers[branch], peer)
	conId, err := s.manager.GetContainerId(branch, 0, false)
	s.Require().NoError(err)

	// fund & activate all accounts
	err = helpers.ExecBankMultiSend(s.manager, &helpers.NodeInfo{Branch: branch}, "dev0", "10aguru", accounts)
	s.Require().NoError(err)
	s.manager.WaitForNextBlock(ctx, conId)

	// copy the genesis file
	genesisFile := fmt.Sprintf("./docker/genesis_%s.json", branch)
	s.Require().NoError(s.manager.CopyFileFromContainer(ctx, conId, "/root/.gurud/config/genesis.json", genesisFile))

	// build a new image and copy the genesis file
	err = s.manager.BuildImage(
		repositoryFullNode,
		branch,
		repoDockerFileFull,
		".",
		map[string]string{
			"BRANCH_NAME": branch,
		},
	)
	s.Require().NoError(err, "can't build container for e2e test")

	// run other validators
	for i := 1; i < nVals; i++ {
		s.T().Log("running validator", i)
		peer := s.runValidatorNode(ctx, branch, strings.Join(peers[branch], ","), i)
		peers[branch] = append(peers[branch], peer)
	}

	// run run full nodes
	for i := 0; i < nFulls; i++ {
		s.T().Log("running full node", i)
		s.runFullNode(ctx, branch, strings.Join(peers[branch], ","), i)
	}

}

func (s *IntegrationTestSuite) runValidatorNode(ctx context.Context, branch, peers string, index int) string {
	// step 1: run as a full node
	options := &dockertest.RunOptions{
		Name:       "validator_node_" + strconv.Itoa(index),
		Repository: repositoryFullNode,
		Tag:        branch,
		Networks:   []*dockertest.Network{s.manager.Networks[branch]},
		Cmd:        []string{"bash", "./init-full-node.sh", peers, "V", strconv.Itoa(index), branch},
	}
	peer := s.runNewNode(ctx, options, branch, true, index)

	// step 2: add new validator
	err := s.createValidatorTx(branch, index)
	s.Require().NoError(err)
	return peer
}

func (s *IntegrationTestSuite) runFullNode(ctx context.Context, branch, peers string, index int) string {
	options := &dockertest.RunOptions{
		Name:       "full_node_" + strconv.Itoa(index),
		Repository: repositoryFullNode,
		Tag:        branch,
		Networks:   []*dockertest.Network{s.manager.Networks[branch]},
		Cmd:        []string{"bash", "./init-full-node.sh", peers, "F", strconv.Itoa(index), branch},
	}
	return s.runNewNode(ctx, options, branch, false, index)
}

func (s *IntegrationTestSuite) runNewNode(ctx context.Context, options *dockertest.RunOptions, branch string, val bool, index int) string {
	var err error
	node := docker.NewNodeWithOptions(options)
	node.SetEnvVars([]string{fmt.Sprintf("CHAIN_ID=%s", chainID)})

	err = s.manager.RunNode(node, val)
	s.Require().NoError(err, "can't run node Guru using branch %s", branch)

	conId, err := s.manager.GetContainerId(branch, index, !val)
	s.Require().NoError(err)
	nodeId, err := s.manager.GetNodeId(ctx, conId)
	s.Require().NoError(err)

	return fmt.Sprintf("%s@%s:26656", nodeId, options.Name)
}

func (s *IntegrationTestSuite) runNetworkIfNotExist(ctx context.Context, node *helpers.NodeInfo) {
	if s.manager.Networks[node.Branch] == nil {
		s.runNetworkWithBranch(ctx, node)
	} else {
		valCount := s.manager.GetValidatorCount(node.Branch)
		fullCount := s.manager.GetFullnodeCount(node.Branch)

		// run the remaining validators
		for i := valCount; i < node.Vals; i++ {
			s.T().Log("running validator", i)
			peer := s.runValidatorNode(ctx, node.Branch, strings.Join(peers[node.Branch], ","), i)
			peers[node.Branch] = append(peers[node.Branch], peer)
		}

		// run the remaining full nodes
		for i := fullCount; i < node.Fulls; i++ {
			s.T().Log("running full node", i)
			s.runFullNode(ctx, node.Branch, strings.Join(peers[node.Branch], ","), i)
		}
	}
}

func (s *IntegrationTestSuite) createValidatorTx(branch string, index int) error {
	nodeInfo := helpers.NodeInfo{Branch: branch, RunOnIndex: index}

	nodeId, err := helpers.QueryNodeId(s.manager, &nodeInfo)
	s.Require().NoError(err)

	pubKey, err := helpers.QueryValidatorPubKey(s.manager, &nodeInfo)
	s.Require().NoError(err)

	err = helpers.ExecCreateValidatorfunc(s.manager, &nodeInfo, "dev0", "validator_node_"+strconv.Itoa(index), "1000000000000000000000aguru", nodeId, pubKey)
	s.Require().NoError(err)

	return nil
}
