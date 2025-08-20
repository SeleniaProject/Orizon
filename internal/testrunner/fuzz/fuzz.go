package fuzz

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"sync/atomic"
	"time"
)

// CorpusEntry represents a single seed input.
type CorpusEntry []byte

// Mutator produces a mutated payload from a parent.
type Mutator func(r *rand.Rand, in []byte) []byte

// Target is the fuzz target. Returning an error or panicking indicates a crash.
type Target func(data []byte) error

// Options controls the fuzzing loop.
type Options struct {
	Duration          time.Duration // total fuzz time
	Seed              int64         // seed for PRNG
	MaxInput          int           // max input size
	Concurrency       int           // parallel workers
	InputBudget       time.Duration // optional per-input budget (0=none)
	MutationIntensity float64       // mutation intensity factor (1.0=default). <=0 uses default
	AutoTune          bool          // adapt intensity based on crash rate
	MaxExecs          uint64        // optional cap on total executions across workers (0=unlimited)
}

// Stats captures aggregate counters for a fuzzing run.
type Stats struct {
	Executions uint64
	Crashes    uint64
}

// DefaultMutator implements a simple byte-level mutation strategy.
func DefaultMutator() Mutator {
	return func(r *rand.Rand, in []byte) []byte {
		out := append([]byte(nil), in...)
		if len(out) == 0 || r.Intn(3) == 0 {
			// insert.
			pos := r.Intn(len(out) + 1)
			b := byte(r.Intn(256))
			out = append(out[:pos], append([]byte{b}, out[pos:]...)...)
		} else if r.Intn(2) == 0 {
			// flip or replace.
			pos := r.Intn(len(out))

			if r.Intn(2) == 0 {
				out[pos] ^= 1 << uint(r.Intn(8))
			} else {
				out[pos] = byte(r.Intn(256))
			}
		} else if len(out) > 0 {
			// delete.
			pos := r.Intn(len(out))
			out = append(out[:pos], out[pos+1:]...)
		}

		return out
	}
}

// AdaptiveMutator returns a mutator that scales the number and aggressiveness of.
// edits based on an atomically adjustable intensity level (percent scale).
func AdaptiveMutator(level *atomic.Uint64) Mutator {
	return func(r *rand.Rand, in []byte) []byte {
		out := append([]byte(nil), in...)
		// intensity: 100 => baseline 1-2 edits, 200 => up to ~3, 300 => up to ~4.
		lv := int(level.Load())
		if lv < 50 {
			lv = 50
		}

		if lv > 300 {
			lv = 300
		}

		maxEdits := 1 + lv/100
		if maxEdits < 1 {
			maxEdits = 1
		}

		if maxEdits > 4 {
			maxEdits = 4
		}

		edits := 1 + r.Intn(maxEdits)
		for i := 0; i < edits; i++ {
			if len(out) == 0 || r.Intn(3) == 0 {
				// insert.
				pos := r.Intn(len(out) + 1)
				b := byte(r.Intn(256))
				out = append(out[:pos], append([]byte{b}, out[pos:]...)...)
			} else if r.Intn(2) == 0 {
				// flip or replace (more bits flipped when intensity is high).
				pos := r.Intn(len(out))

				if r.Intn(2) == 0 {
					flips := 1 + r.Intn(1+lv/120) // up to 3 bits at high intensity
					for k := 0; k < flips; k++ {
						out[pos] ^= 1 << uint(r.Intn(8))
					}
				} else {
					out[pos] = byte(r.Intn(256))
				}
			} else if len(out) > 0 {
				// delete small span at higher intensity.
				pos := r.Intn(len(out))
				span := 1

				if lv >= 200 && len(out)-pos > 2 {
					span = 1 + r.Intn(min(3, len(out)-pos))
				}

				out = append(out[:pos], out[pos+span:]...)
			}
		}
		// clamp to avoid excessive growth.
		return out
	}
}

// Run executes a time-bounded fuzzing campaign.
func Run(opts Options, corpus []CorpusEntry, target Target, mut Mutator, crashes io.Writer) {
	_ = RunWithStats(opts, corpus, target, mut, crashes)
}

// RunWithStats is like Run, but returns aggregate counters.
func RunWithStats(opts Options, corpus []CorpusEntry, target Target, mut Mutator, crashes io.Writer) Stats {
	if opts.Duration <= 0 {
		opts.Duration = 3 * time.Second
	}

	if opts.Seed == 0 {
		opts.Seed = time.Now().UnixNano()
	}

	if opts.MaxInput <= 0 {
		opts.MaxInput = 1 << 12
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	// Defer mutator selection until after intensity/autotune configuration

	stop := time.Now().Add(opts.Duration)

	type job struct{ data []byte }

	jobs := make(chan job, 1024)
	// enqueue initial corpus.
	go func() {
		for _, c := range corpus {
			if len(c) > 0 {
				jobs <- job{data: append([]byte(nil), c...)}
			}
		}
		// synthetic seed.
		jobs <- job{data: []byte("ORIZON-FUZZ-SEED")}
	}()

	var execCount uint64

	var crashCount uint64

	var quit uint32
	// configure mutator (adaptive if requested).
	var level atomic.Uint64
	if opts.MutationIntensity > 0 {
		level.Store(uint64(opts.MutationIntensity * 100))
	} else {
		level.Store(100)
	}

	if mut == nil {
		if opts.AutoTune || opts.MutationIntensity > 0 {
			mut = AdaptiveMutator(&level)
		} else {
			mut = DefaultMutator()
		}
	}

	for w := 0; w < opts.Concurrency; w++ {
		r := rand.New(rand.NewSource(derive(opts.Seed, w)))
		go func(r *rand.Rand) {
			cur := []byte("ORIZON")

			for time.Now().Before(stop) {
				if atomic.LoadUint32(&quit) == 1 {
					return
				}
				select {
				case j := <-jobs:
					cur = j.data
				default:
				}

				cand := mut(r, cur)
				if len(cand) > opts.MaxInput {
					cand = cand[:opts.MaxInput]
				}

				var err error

				if opts.InputBudget > 0 {
					ch := make(chan error, 1)
					go func(d []byte) { ch <- callTargetSafe(target, d) }(cand)
					select {
					case e := <-ch:
						err = e
					case <-time.After(opts.InputBudget):
						err = io.EOF // report as failure to trigger crash output
					}
				} else {
					err = callTargetSafe(target, cand)
				}

				newExec := atomic.AddUint64(&execCount, 1)

				if err != nil {
					atomic.AddUint64(&crashCount, 1)

					if crashes != nil {
						// Encode input safely as hex to keep crash logs line-oriented and parseable.
						_, _ = crashes.Write([]byte(time.Now().Format(time.RFC3339Nano) + "\t"))
						encoded := hex.EncodeToString(cand)
						_, _ = crashes.Write([]byte("0x" + encoded))
						_, _ = crashes.Write([]byte("\t" + err.Error() + "\n"))
					}
					// optional autotune: increase intensity slightly when crash rate is very low, decrease when high.
					if opts.AutoTune {
						execs := atomic.LoadUint64(&execCount)
						crashesNow := atomic.LoadUint64(&crashCount)

						if execs%1000 == 0 && execs > 0 { // adjust periodically
							rate := float64(crashesNow) / float64(execs)
							curLv := level.Load()

							if rate < 0.0005 && curLv < 300 {
								level.Store(curLv + 10)
							} else if rate > 0.01 && curLv > 80 {
								level.Store(curLv - 10)
							}
						}
					}
				}

				if opts.MaxExecs > 0 && newExec >= opts.MaxExecs {
					atomic.StoreUint32(&quit, 1)

					return
				}

				cur = cand
			}
		}(r)
	}
	// Wait until duration elapses or quit is triggered.
	deadline := time.Now().Add(opts.Duration)
	for time.Now().Before(deadline) {
		if atomic.LoadUint32(&quit) == 1 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	return Stats{Executions: atomic.LoadUint64(&execCount), Crashes: atomic.LoadUint64(&crashCount)}
}

// callTargetSafe invokes the target and converts panics into errors for recording.
func callTargetSafe(t Target, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return t(data)
}

func derive(base int64, salt int) int64 {
	var b [16]byte

	binary.LittleEndian.PutUint64(b[0:8], uint64(base))
	binary.LittleEndian.PutUint64(b[8:16], uint64(salt))
	sh := sha256.Sum256(b[:])

	return int64(binary.LittleEndian.Uint64(sh[:8]))
}

// Minimize attempts to reduce input while preserving failure (target returns non-nil).
// It applies a greedy delta-debugging inspired process within the given time budget.
func Minimize(seed int64, in []byte, target Target, budget time.Duration) []byte {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	r := rand.New(rand.NewSource(seed))
	start := time.Now()

	best := append([]byte(nil), in...)
	if target(best) == nil {
		return best
	}
	// Strategies: delete chunks, flip bits, replace bytes, shrink tail.
	for time.Since(start) < budget {
		progressed := false
		// Try removing halves and quarters.
		for parts := 2; parts <= 8 && time.Since(start) < budget; parts *= 2 {
			n := len(best)
			if n < parts {
				break
			}

			seg := n / parts
			for i := 0; i < parts && time.Since(start) < budget; i++ {
				cand := append([]byte(nil), best[:i*seg]...)
				cand = append(cand, best[(i+1)*seg:]...)

				if len(cand) == 0 {
					continue
				}

				if target(cand) != nil {
					best = cand
					progressed = true

					break
				}
			}

			if progressed {
				break
			}
		}

		if progressed {
			continue
		}
		// Try truncating tail.
		if len(best) > 1 {
			cand := append([]byte(nil), best[:len(best)-1]...)
			if target(cand) != nil {
				best = cand

				continue
			}
		}
		// Bit flips and byte replacement.
		if len(best) > 0 {
			idx := r.Intn(len(best))
			// flip one bit.
			b := best[idx]
			cand := append([]byte(nil), best...)
			cand[idx] = b ^ (1 << uint(r.Intn(8)))

			if target(cand) != nil {
				best = cand

				continue
			}
			// replace.
			cand[idx] = byte(r.Intn(256))
			if target(cand) != nil {
				best = cand

				continue
			}
		}
		// If no progress, random deletion.
		if len(best) > 1 {
			i := r.Intn(len(best))
			cand := append([]byte(nil), best[:i]...)
			cand = append(cand, best[i+1:]...)

			if len(cand) > 0 && target(cand) != nil {
				best = cand

				continue
			}
		}

		break
	}

	return best
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
