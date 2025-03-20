package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

const ()

type IntegrationTestSuite struct {
	suite.Suite
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
