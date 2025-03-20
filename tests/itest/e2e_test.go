package itest

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/GPTx-global/guru/tests/itest/helpers"
)

// TestCLITxs executes different types of transactions against an Evmos node
// using the CLI client. The node used for the test has the latest changes introduced.
func (s *IntegrationTestSuite) TestCLITxs() {
	for _, tc := range helpers.TestCases {
		s.Run(fmt.Sprintf("%s :: %s", tc.Module, tc.Name), func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Reset the node if needed
			if tc.Reset || len(tc.Node.GenesisParams) > 0 {
				s.T().Log("Resetting the network with branch ", tc.Node.Branch)
				s.runNetworkWithBranch(ctx, &tc.Node)
			} else {
				// run node if not exist yet
				s.runNetworkIfNotExist(ctx, &tc.Node)
			}

			// execute the pre-state changes
			preArgs, err := tc.Malleate_pre(ctx, s.manager)
			s.Require().NoError(err)

			cmd := tc.Cmd(preArgs)
			if len(cmd) > 0 {
				conId, err := s.manager.GetContainerId(tc.Node.Branch, tc.Node.RunOnIndex, tc.Node.RunOnFullnode)
				s.Require().NoError(err)

				exec, err := s.manager.CreateExec(cmd, conId)
				s.Require().NoError(err)

				outBuf, errBuf, err := s.manager.RunExec(ctx, exec)
				s.Require().NoError(err)

				s.manager.WaitForNextBlock(ctx, conId)

				// fmt.Println("OUT: ", outBuf.String())
				if tc.ExpPass {
					s.Require().Truef(
						strings.Contains(outBuf.String(), "code: 0"),
						"tx returned non code 0:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String(),
					)
				} else {
					s.Require().NotContains(outBuf.String(), "code: 0")
					s.Require().Truef(
						strings.Contains(outBuf.String(), tc.ExpErr) || strings.Contains(errBuf.String(), tc.ExpErr),
						"tx returned non code 0 but with unexpected error:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String(),
					)
				}

				// test for post execution cases
				for _, check := range tc.CheckCases {
					queryExec, err := s.manager.CreateQueryExec(conId, tc.Node.Branch, check.Module, check.Query, check.Args...)
					s.Require().NoError(err)
					outBuf, errBuf, err := s.manager.RunExec(ctx, queryExec)
					s.Require().NoError(err)
					s.Require().Empty(errBuf.String())

					s.Require().Equal(outBuf.String(), check.Expected)
				}
			}
			s.Require().NoError(tc.Malleate_post(ctx, s.manager, preArgs))
		})
	}

}
