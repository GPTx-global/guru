package helpers

import (
	"context"

	"github.com/GPTx-global/guru/tests/itest/docker"
)

type CheckCase struct {
	Module   string   // which module to query from?
	Query    string   // query command
	Args     []string // query args
	Expected string   // expected output of query
}

type NodeInfo struct {
	Branch        string   // branch name
	GenesisParams []string // custom genesis params (network is reset)
	Vals          int      // numbe rof validators in the network
	Fulls         int      // numbe rof full/sentry nodes
	RunOnFullnode bool     // execute the cmd on validator or on full node? default: validator
	RunOnIndex    int      // execute the cmd on specific node. default: 0
}

type TestCase struct {
	Reset         bool                                                                            // whether to reset the docker container
	Node          NodeInfo                                                                        // Network & Node config
	Module        string                                                                          // name of module
	Name          string                                                                          // test case
	Malleate_pre  func(ctx context.Context, m *docker.Manager) (map[string]interface{}, error)    // change the state before execution of Cmd
	Cmd           func(map[string]interface{}) []string                                           // command to execute
	ExpPass       bool                                                                            // should pass or not?
	ExpErr        string                                                                          // expected error (only if ExpPass == false)
	CheckCases    []CheckCase                                                                     // test cases
	Malleate_post func(ctx context.Context, m *docker.Manager, args map[string]interface{}) error // change & check the state after execution of Cmd
}

var TestCases []TestCase

func AddTestCase(testCase *TestCase) {
	if testCase.Node.Branch == "" {
		testCase.Node.Branch = "main"
	}
	TestCases = append(TestCases, *testCase)
}
