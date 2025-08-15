package prop

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"runtime"
	"time"
)

// Generator produces a value of type T from a PRNG and a size hint.
type Generator[T any] func(r *rand.Rand, size int) T

// Shrinker produces a slice of candidate smaller values that aim to preserve failure.
type Shrinker[T any] func(v T) []T

// Property1 is a unary property predicate.
type Property1[A any] func(a A) bool

// Options control property checking.
type Options struct {
	Trials          int           // number of trials
	Seed            int64         // random seed; 0 means time.Now().UnixNano()
	Size            int           // size hint for generators
	Parallelism     int           // number of workers; <=0 means GOMAXPROCS
	MaxShrinkRounds int           // limit for shrinking attempts
	MaxShrinkTime   time.Duration // wall time limit for shrinking; 0 to disable
}

// Result is the outcome of a property check.
type Result struct {
	PassedTrials int
	Failed       bool
	FailingInput any
	ShrunkInput  any
	Seed         int64
	Duration     time.Duration
	ShrinkRounds int
}

// ForAll1 checks a unary property with the provided generator and optional shrinker.
func ForAll1[A any](genA Generator[A], shrinkA Shrinker[A], prop Property1[A], opts Options) Result {
	start := time.Now()
	if opts.Trials <= 0 {
		opts.Trials = 200
	}
	if opts.Seed == 0 {
		opts.Seed = time.Now().UnixNano()
	}
	if opts.Size <= 0 {
		opts.Size = 30
	}
	if opts.Parallelism <= 0 {
		opts.Parallelism = runtime.GOMAXPROCS(0)
		if opts.Parallelism <= 0 {
			opts.Parallelism = 1
		}
	}
	if opts.MaxShrinkRounds <= 0 {
		opts.MaxShrinkRounds = 200
	}
	// worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type task struct{ idx int }
	type outcome struct {
		idx int
		a   A
		ok  bool
	}
	tasks := make(chan task)
	outs := make(chan outcome)

	// launch workers
	for w := 0; w < opts.Parallelism; w++ {
		go func(worker int) {
			for t := range tasks {
				// derive deterministic seed per trial and worker
				r := rand.New(rand.NewSource(deriveSeed(opts.Seed, t.idx)))
				a := genA(r, opts.Size)
				ok := prop(a)
				select {
				case outs <- outcome{idx: t.idx, a: a, ok: ok}:
				case <-ctx.Done():
					return
				}
			}
		}(w)
	}

	// feed tasks
	go func() {
		for i := 0; i < opts.Trials; i++ {
			select {
			case tasks <- task{idx: i}:
			case <-ctx.Done():
				close(tasks)
				return
			}
		}
		close(tasks)
	}()

	var res Result
	res.Seed = opts.Seed
	for completed := 0; completed < opts.Trials; completed++ {
		o := <-outs
		if o.ok {
			res.PassedTrials++
			continue
		}
		// first failure: stop generating and shrink
		res.Failed = true
		res.FailingInput = o.a
		cancel()
		// Shrink synchronously
		if shrinkA != nil {
			deadline := time.Time{}
			if opts.MaxShrinkTime > 0 {
				deadline = time.Now().Add(opts.MaxShrinkTime)
			}
			best := o.a
			rounds := 0
			for {
				if opts.MaxShrinkTime > 0 && time.Now().After(deadline) {
					break
				}
				if rounds >= opts.MaxShrinkRounds {
					break
				}
				candidates := shrinkA(best)
				if len(candidates) == 0 {
					break
				}
				progressed := false
				for _, c := range candidates {
					if !prop(c) {
						best = c
						progressed = true
						break
					}
				}
				rounds++
				if !progressed {
					break
				}
			}
			res.ShrunkInput = best
			res.ShrinkRounds = rounds
		}
		break
	}
	res.Duration = time.Since(start)
	return res
}

// deriveSeed deterministically mixes base seed with trial index via SHA-256.
func deriveSeed(base int64, idx int) int64 {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], uint64(base))
	binary.LittleEndian.PutUint64(b[8:16], uint64(idx))
	h := sha256.Sum256(b[:])
	return int64(binary.LittleEndian.Uint64(h[0:8]))
}
