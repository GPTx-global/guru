package e2e

import (
	"context"
	"fmt"
	"strings"

	cmd "github.com/GPTx-global/guru/tests/e2e/cmd"
)

// TestCLITxs executes different types of transactions against an Evmos node
// using the CLI client. The node used for the test has the latest changes introduced.
func (s *IntegrationTestSuite) TestCLITxs() {
	for _, tc := range cmd.TestCases {
		s.Run(fmt.Sprintf("%s :: %s", tc.Module, tc.Name), func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Reset the node if needed
			if tc.Reset {
				s.runNodeWithBranch(tc.Branch)
			}
			// run node if not exist yet
			s.runNodeIfNotExist(tc.Branch)

			exec, err := s.manager.CreateExec(tc.Cmd, s.manager.ContainerID(tc.Branch))
			s.Require().NoError(err)

			outBuf, errBuf, err := s.manager.RunExec(ctx, exec)
			s.Require().NoError(err)

			// fmt.Println("OUT: ", outBuf.String())
			if tc.ExpPass {
				s.Require().Truef(
					strings.Contains(outBuf.String(), "code: 0"),
					"tx returned non code 0:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String(),
				)
				for _, check := range tc.PassCheck {
					queryExec, err := s.manager.CreateQueryExec(tc.Branch, check.Module, check.Query, check.Args...)
					s.Require().NoError(err)
					outBuf, errBuf, err := s.manager.RunExec(ctx, queryExec)
					s.Require().NoError(err)
					s.Require().Empty(errBuf.String())

					s.Require().Equal(outBuf.String(), check.Expected)
				}
				s.Require().NoError(tc.PassCheckFunc())
			} else {
				s.Require().NotContains(outBuf.String(), "code: 0")
				s.Require().Truef(
					strings.Contains(outBuf.String(), tc.ExpErr) || strings.Contains(errBuf.String(), tc.ExpErr),
					"tx returned non code 0 but with unexpected error:\nstdout: %s\nstderr: %s", outBuf.String(), errBuf.String(),
				)
				for _, check := range tc.FailCheck {
					queryExec, err := s.manager.CreateQueryExec(tc.Branch, check.Module, check.Query, check.Args...)
					s.Require().NoError(err)
					outBuf, errBuf, err := s.manager.RunExec(ctx, queryExec)
					s.Require().NoError(err)
					s.Require().Empty(errBuf.String())

					s.Require().Equal(outBuf.String(), check.Expected)
				}
				s.Require().NoError(tc.FailCheckFunc())
			}
		})
	}

}
