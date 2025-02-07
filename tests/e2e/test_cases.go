package e2e

type CheckCase struct {
	Module   string   // which module to query from?
	Query    string   // query command
	Args     []string // query args
	Expected string   // expected output of query
}

type TestCase struct {
	Reset         bool     // whether to reset the docker container
	Branch        string   // which branch to run the test on
	Module        string   // name of module
	Name          string   // test case
	Cmd           []string // command to execute
	ExpPass       bool     // should pass or not?
	ExpErr        string   // expected error (only if ExpPass == false)
	PassCheck     []CheckCase
	FailCheck     []CheckCase
	PassCheckFunc func() error // conditions to check in case of ExpPass == true
	FailCheckFunc func() error // conditions to check in case of ExpPass == false
}

var TestCases []TestCase

func AddTestCase(testCase *TestCase) {
	if testCase.Branch == "" {
		testCase.Branch = "main"
	}
	TestCases = append(TestCases, *testCase)
}
