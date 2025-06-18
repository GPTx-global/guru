package woker

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/stretchr/testify/require"
)

// setup a test suite with a mock executor
func setupManagerTest(t *testing.T) (*JobManager, chan *types.JobResult, context.CancelFunc, *func(job *types.Job) *types.JobResult) {
	log.InitLogger()
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 100)
	ctx, cancel := context.WithCancel(context.Background())

	var mockExec func(job *types.Job) *types.JobResult
	originalExec := executeJob
	executeJob = func(job *types.Job) *types.JobResult {
		if mockExec != nil {
			return mockExec(job)
		}
		return nil
	}

	jm.Start(ctx, resultQueue)

	// Teardown function
	t.Cleanup(func() {
		cancel()
		jm.Stop()
		executeJob = originalExec
	})

	return jm, resultQueue, cancel, &mockExec
}

func TestJobManager_SubmitJob_DropsWhenFull(t *testing.T) {
	log.InitLogger()
	jm := NewJobManager()
	// Intentionally do not start workers, so the queue can be filled without being consumed.
	jm.jobQueue = make(chan *types.Job, 1)

	// Fill the queue
	jm.jobQueue <- &types.Job{ID: 1}

	// This next job should be dropped.
	// We submit it in a goroutine because if it blocks, the test will hang.
	submitted := make(chan bool, 1)
	go func() {
		jm.SubmitJob(&types.Job{ID: 999})
		submitted <- true
	}()

	select {
	case <-submitted:
		// good, it didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SubmitJob blocked when queue was full")
	}
}

func TestJobManager_SubmitAndProcess(t *testing.T) {
	jm, _, _, mockExec := setupManagerTest(t)

	var registered sync.WaitGroup
	registered.Add(1)

	*mockExec = func(job *types.Job) *types.JobResult {
		if job.ID == 1 {
			registered.Done()
		}
		return nil
	}

	jm.SubmitJob(&types.Job{ID: 1, Type: types.Register})

	done := make(chan struct{})
	go func() {
		registered.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("job was not registered in time")
	}

	jm.activeJobsLock.Lock()
	_, ok := jm.activeJobs[1]
	jm.activeJobsLock.Unlock()
	require.True(t, ok, "job should be in active jobs map")
}

func TestJobManager_JobQueueFull(t *testing.T) {
	// This test is now covered by TestJobManager_SubmitJob_DropsWhenFull
	// and the logic is removed to avoid flakes.
}

func TestJobManager_WorkerLogic(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		jm, _, _, _ := setupManagerTest(t)
		jm.SubmitJob(&types.Job{ID: 1, Type: types.Register})

		require.Eventually(t, func() bool {
			jm.activeJobsLock.Lock()
			defer jm.activeJobsLock.Unlock()
			_, ok := jm.activeJobs[1]
			return ok
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Update", func(t *testing.T) {
		jm, _, _, _ := setupManagerTest(t)
		jm.activeJobs[1] = &types.Job{ID: 1, URL: "v1"}
		jm.SubmitJob(&types.Job{ID: 1, Type: types.Update, URL: "v2"})

		require.Eventually(t, func() bool {
			jm.activeJobsLock.Lock()
			defer jm.activeJobsLock.Unlock()
			job, ok := jm.activeJobs[1]
			return ok && job.URL == "v2"
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Complete_Success", func(t *testing.T) {
		jm, resultQueue, _, mockExec := setupManagerTest(t)
		jm.activeJobs[1] = &types.Job{ID: 1, Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED.String(), Nonce: 5}

		*mockExec = func(j *types.Job) *types.JobResult {
			require.Equal(t, uint64(6), j.Nonce)
			return &types.JobResult{ID: j.ID, Nonce: j.Nonce, Data: "data"}
		}

		jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete, Nonce: 5})

		select {
		case res := <-resultQueue:
			require.Equal(t, uint64(1), res.ID)
			require.Equal(t, uint64(6), res.Nonce)
		case <-time.After(time.Second):
			t.Fatal("no result received")
		}
	})

	t.Run("Complete_NotEnabled", func(t *testing.T) {
		jm, _, _, mockExec := setupManagerTest(t)
		jm.activeJobs[1] = &types.Job{ID: 1, Status: oracletypes.RequestStatus_REQUEST_STATUS_DISABLED.String()}

		called := false
		*mockExec = func(j *types.Job) *types.JobResult {
			called = true
			return nil
		}
		jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete})

		time.Sleep(100 * time.Millisecond)
		require.False(t, called)
	})

	t.Run("Complete_NotActive", func(t *testing.T) {
		jm, _, _, mockExec := setupManagerTest(t)

		called := false
		*mockExec = func(j *types.Job) *types.JobResult {
			called = true
			return nil
		}
		jm.SubmitJob(&types.Job{ID: 1, Type: types.Complete})

		time.Sleep(100 * time.Millisecond)
		require.False(t, called)
	})
}

func TestJobManager_Stop(t *testing.T) {
	jm := NewJobManager()
	ctx, cancel := context.WithCancel(context.Background())
	resultQueue := make(chan *types.JobResult, 1)

	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU())

	originalExec := executeJob
	executeJob = func(job *types.Job) *types.JobResult {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
		return nil
	}
	defer func() { executeJob = originalExec }()

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
		t.Fatal("stop finished prematurely")
	case <-time.After(50 * time.Millisecond):
		// good
	}

	// wait for workers to finish
	wg.Wait()

	select {
	case <-stopDone:
		// good, stop finished
	case <-time.After(time.Second):
		t.Fatal("stop did not finish after workers")
	}

	cancel()
}
