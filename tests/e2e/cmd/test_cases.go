package cmd

type CheckCase struct {
	Module   string   // which module to query from?
	Query    string   // query command
	Args     []string // query args
	Expected string   // expected output of query
}

type TestCase struct {
	Reset         bool         // whether to reset the docker container
	Branch        string       // which branch to run the test on
	Module        string       // name of module
	Name          string       // test case
	Malleate_pre  func() error // change the state before execution of Cmd
	Cmd           []string     // command to execute
	ExpPass       bool         // should pass or not?
	ExpErr        string       // expected error (only if ExpPass == false)
	CheckCases    []CheckCase  // test cases
	Malleate_post func() error // change & check the state after execution of Cmd
}

var TestCases []TestCase

func AddTestCase(testCase *TestCase) {
	if testCase.Branch == "" {
		testCase.Branch = "main"
	}
	TestCases = append(TestCases, *testCase)
}
