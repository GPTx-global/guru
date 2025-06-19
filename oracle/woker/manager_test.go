package woker

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/suite"
)

type ManagerSuite struct {
	suite.Suite
	jm           *JobManager
	resultQueue  chan *types.JobResult
	cancel       context.CancelFunc
	mockExec     func(job *types.Job) (*types.JobResult, error)
	originalExec func(job *types.Job) (*types.JobResult, error)
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

func (s *ManagerSuite) SetupTest() {
	log.InitLogger()
	config.SetForTesting("test-chain", "", "test", os.TempDir(), keyring.BackendTest, "", 0, 3)

	s.jm = NewJobManager()
	s.resultQueue = make(chan *types.JobResult, 100)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.originalExec = executeJob
	executeJob = func(job *types.Job) (*types.JobResult, error) {
		if s.mockExec != nil {
			return s.mockExec(job)
		}
		return nil, nil
	}

	s.jm.Start(ctx, s.resultQueue)
}

func (s *ManagerSuite) TearDownTest() {
	s.cancel()
	s.jm.Stop()
	executeJob = s.originalExec
	s.mockExec = nil
}

func (s *ManagerSuite) TestSubmitJob_DropsWhenFull() {
	// This test requires a specific setup (unstarted manager), so we don't use the suite's s.jm
	jm := NewJobManager()
	jm.jobQueue = make(chan *types.Job, 1)

	// Fill the queue
	jm.jobQueue <- &types.Job{ID: 1}

	// This next job should be dropped.
	submitted := make(chan bool, 1)
	go func() {
		jm.SubmitJob(&types.Job{ID: 999})
		submitted <- true
	}()

	select {
	case <-submitted:
		// good, it didn't block
	case <-time.After(100 * time.Millisecond):
		s.T().Fatal("SubmitJob blocked when queue was full")
	}
}

func (s *ManagerSuite) TestSubmitAndProcess() {
	var registered sync.WaitGroup
	registered.Add(1)

	s.mockExec = func(job *types.Job) (*types.JobResult, error) {
		if job.ID == 1 {
			registered.Done()
		}
		return nil, nil
	}

	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Register})

	done := make(chan struct{})
	go func() {
		registered.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		s.T().Fatal("job was not registered in time")
	}

	s.jm.activeJobsLock.Lock()
	_, ok := s.jm.activeJobs[1]
	s.jm.activeJobsLock.Unlock()
	s.Require().True(ok, "job should be in active jobs map")
}

func (s *ManagerSuite) TestWorkerLogic_Register() {
	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Register})

	s.Require().Eventually(func() bool {
		s.jm.activeJobsLock.Lock()
		defer s.jm.activeJobsLock.Unlock()
		_, ok := s.jm.activeJobs[1]
		return ok
	}, time.Second, 10*time.Millisecond)
}

func (s *ManagerSuite) TestWorkerLogic_Update() {
	s.jm.activeJobs[1] = &types.Job{ID: 1, URL: "v1"}
	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Update, URL: "v2"})

	s.Require().Eventually(func() bool {
		s.jm.activeJobsLock.Lock()
		defer s.jm.activeJobsLock.Unlock()
		job, ok := s.jm.activeJobs[1]
		return ok && job.URL == "v2"
	}, time.Second, 10*time.Millisecond)
}

func (s *ManagerSuite) TestWorkerLogic_Complete_Success() {
	s.jm.activeJobs[1] = &types.Job{ID: 1, Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED.String(), Nonce: 5}

	s.mockExec = func(j *types.Job) (*types.JobResult, error) {
		s.Require().Equal(uint64(6), j.Nonce)
		return &types.JobResult{ID: j.ID, Nonce: j.Nonce, Data: "data"}, nil
	}

	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete, Nonce: 5})

	select {
	case res := <-s.resultQueue:
		s.Require().Equal(uint64(1), res.ID)
		s.Require().Equal(uint64(6), res.Nonce)
	case <-time.After(time.Second):
		s.T().Fatal("no result received")
	}
}

func (s *ManagerSuite) TestWorkerLogic_Complete_NotEnabled() {
	s.jm.activeJobs[1] = &types.Job{ID: 1, Status: oracletypes.RequestStatus_REQUEST_STATUS_DISABLED.String()}

	called := false
	s.mockExec = func(j *types.Job) (*types.JobResult, error) {
		called = true
		return nil, nil
	}
	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete})

	time.Sleep(100 * time.Millisecond)
	s.Require().False(called)
}

func (s *ManagerSuite) TestWorkerLogic_Complete_NotActive() {
	called := false
	s.mockExec = func(j *types.Job) (*types.JobResult, error) {
		called = true
		return nil, nil
	}
	s.jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete})

	time.Sleep(100 * time.Millisecond)
	s.Require().False(called)
}

func (s *ManagerSuite) TestStop() {
	// This test has its own manager lifecycle, so it doesn't use the suite's jm.
	jm := NewJobManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultQueue := make(chan *types.JobResult, 1)

	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU())

	// Use local mock for this test
	original := executeJob
	executeJob = func(job *types.Job) (*types.JobResult, error) {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
		return nil, nil
	}
	defer func() { executeJob = original }()

	jm.Start(ctx, resultQueue)
	for i := 0; i < runtime.NumCPU(); i++ {
		jm.SubmitJob(&types.Job{ID: uint64(i)})
	}

	stopDone := make(chan struct{})
	go func() {
		jm.Stop()
		close(stopDone)
	}()

	// check that stop is blocking
	select {
	case <-stopDone:
		s.T().Fatal("stop finished prematurely")
	case <-time.After(50 * time.Millisecond):
		// good
	}

	// wait for workers to finish
	wg.Wait()

	select {
	case <-stopDone:
		// good, stop finished
	case <-time.After(time.Second):
		s.T().Fatal("stop did not finish after workers")
	}
}
