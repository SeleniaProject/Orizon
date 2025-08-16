package concurrency

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Scheduler provides a controllable scheduling environment for exploring
// different interleavings of concurrent tasks. It is inspired by PCT-style
// randomized schedulers and is suitable for small-to-medium concurrency tests.
type Scheduler struct {
	seed    int64
	r       *rand.Rand
	ready   chan *task
	wg      sync.WaitGroup
	stopped atomic.Uint32

	// fairness window for yielding decisions
	quantum int

	doneCh chan struct{}
}

type task struct {
	id   int
	step chan struct{}
	run  func(ctx context.Context, s *Scheduler)
}

// Options configures the Scheduler behavior.
type Options struct {
	Seed    int64
	Quantum int // number of yields between random steals; default 1
}

// New creates a new Scheduler with the specified options.
func New(opts Options) *Scheduler {
	if opts.Seed == 0 {
		opts.Seed = time.Now().UnixNano()
	}
	if opts.Quantum <= 0 {
		opts.Quantum = 1
	}
	return &Scheduler{
		seed:    opts.Seed,
		r:       rand.New(rand.NewSource(opts.Seed)),
		ready:   make(chan *task, 1024),
		quantum: opts.Quantum,
		doneCh:  make(chan struct{}),
	}
}

// Seed returns the scheduler seed.
func (s *Scheduler) Seed() int64 { return s.seed }

// Go registers and starts a new task under scheduler control.
func (s *Scheduler) Go(fn func(ctx context.Context, sched *Scheduler)) {
	t := &task{id: s.r.Int(), step: make(chan struct{}, 1), run: fn}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// first admission requires step token
		s.ready <- t
		<-t.step
		fn(ctx, s)
	}()
}

// Run executes until all tasks finish or the context cancels.
func (s *Scheduler) Run(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}
	// notify when all tasks complete
	go func() { s.wg.Wait(); close(s.doneCh) }()
	// scheduler loop
	yieldCount := 0
	idleStreak := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.doneCh:
			// Drain any residual ready steps for fairness; then exit when queue is quiet
			if drained(s.ready) {
				return nil
			}
		case t := <-s.ready:
			// allow task to make progress, then decide next step
			s.allow(t)
			yieldCount++
			if yieldCount%s.quantum == 0 {
				runtime.Gosched()
			}
		case <-time.After(1 * time.Millisecond):
			// back off to avoid spinning
			idleStreak++
			if idleStreak > 5 {
				// if done and repeatedly idle, exit
				select {
				case <-s.doneCh:
					return nil
				default:
				}
				idleStreak = 0
			}
		}
	}
}

// allow grants a step token to the chosen task.
func (s *Scheduler) allow(t *task) {
	select {
	case t.step <- struct{}{}:
	default:
	}
}

// Yield should be called by tasks to hand control back to the scheduler.
func (s *Scheduler) Yield() {
	if s.stopped.Load() == 1 {
		return
	}
	// requeue current goroutine as a task by recovering its parked token
	// The technique: block until scheduler grants next step token.
	// Callers should ensure cooperative yield points.
	current := &task{step: make(chan struct{}, 1)}
	s.ready <- current
	<-current.step
}

// Park blocks the current task for a duration to model external waiting.
func (s *Scheduler) Park(d time.Duration) { time.Sleep(d) }

// Stop requests the scheduler to cease scheduling new steps.
func (s *Scheduler) Stop() { s.stopped.Store(1) }

// Wait waits for all tasks to finish.
func (s *Scheduler) Wait() { s.wg.Wait() }

// anyRunning checks if worker goroutines are still active.
func drained[T any](ch <-chan T) bool {
	select {
	case <-ch:
		return false
	default:
		return true
	}
}

// Explore runs the provided factory multiple times with different seeds.
// The factory should register tasks and return a function to wait for completion.
func Explore(trials int, factory func(seed int64) (wait func() error)) []error {
	if trials <= 0 {
		trials = runtime.GOMAXPROCS(0)
		if trials < 1 {
			trials = 1
		}
	}
	errs := make([]error, trials)
	var wg sync.WaitGroup
	for i := 0; i < trials; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seed := time.Now().UnixNano() + int64(i)
			wait := factory(seed)
			if wait != nil {
				errs[i] = wait()
			}
		}(i)
	}
	wg.Wait()
	return errs
}
