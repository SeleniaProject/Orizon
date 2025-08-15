package fuzz

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math/rand"
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
	Duration    time.Duration // total fuzz time
	Seed        int64         // seed for PRNG
	MaxInput    int           // max input size
	Concurrency int           // parallel workers
}

// DefaultMutator implements a simple byte-level mutation strategy.
func DefaultMutator() Mutator {
	return func(r *rand.Rand, in []byte) []byte {
		out := append([]byte(nil), in...)
		if len(out) == 0 || r.Intn(3) == 0 {
			// insert
			pos := r.Intn(len(out) + 1)
			b := byte(r.Intn(256))
			out = append(out[:pos], append([]byte{b}, out[pos:]...)...)
		} else if r.Intn(2) == 0 {
			// flip or replace
			pos := r.Intn(len(out))
			if r.Intn(2) == 0 {
				out[pos] ^= 1 << uint(r.Intn(8))
			} else {
				out[pos] = byte(r.Intn(256))
			}
		} else if len(out) > 0 {
			// delete
			pos := r.Intn(len(out))
			out = append(out[:pos], out[pos+1:]...)
		}
		return out
	}
}

// Run executes a time-bounded fuzzing campaign.
func Run(opts Options, corpus []CorpusEntry, target Target, mut Mutator, crashes io.Writer) {
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
	if mut == nil {
		mut = DefaultMutator()
	}

	stop := time.Now().Add(opts.Duration)
	type job struct{ data []byte }
	jobs := make(chan job, 1024)
	// enqueue initial corpus
	go func() {
		for _, c := range corpus {
			if len(c) > 0 {
				jobs <- job{data: append([]byte(nil), c...)}
			}
		}
		// synthetic seed
		jobs <- job{data: []byte("ORIZON-FUZZ-SEED")}
	}()

	for w := 0; w < opts.Concurrency; w++ {
		r := rand.New(rand.NewSource(derive(opts.Seed, w)))
		go func(r *rand.Rand) {
			cur := []byte("ORIZON")
			for time.Now().Before(stop) {
				select {
				case j := <-jobs:
					cur = j.data
				default:
				}
				cand := mut(r, cur)
				if len(cand) > opts.MaxInput {
					cand = cand[:opts.MaxInput]
				}
				if err := target(cand); err != nil {
					if crashes != nil {
						_, _ = crashes.Write([]byte(time.Now().Format(time.RFC3339Nano) + "\t"))
						_, _ = crashes.Write(cand)
						_, _ = crashes.Write([]byte("\t" + err.Error() + "\n"))
					}
				}
				cur = cand
			}
		}(r)
	}
	time.Sleep(opts.Duration)
}

func derive(base int64, salt int) int64 {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], uint64(base))
	binary.LittleEndian.PutUint64(b[8:16], uint64(salt))
	h := sha256.Sum256(b[:])
	return int64(binary.LittleEndian.Uint64(h[:8]))
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
	// Strategies: delete chunks, flip bits, replace bytes, shrink tail
	for time.Since(start) < budget {
		progressed := false
		// Try removing halves and quarters
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
		// Try truncating tail
		if len(best) > 1 {
			cand := append([]byte(nil), best[:len(best)-1]...)
			if target(cand) != nil {
				best = cand
				continue
			}
		}
		// Bit flips and byte replacement
		if len(best) > 0 {
			idx := r.Intn(len(best))
			// flip one bit
			b := best[idx]
			cand := append([]byte(nil), best...)
			cand[idx] = b ^ (1 << uint(r.Intn(8)))
			if target(cand) != nil {
				best = cand
				continue
			}
			// replace
			cand[idx] = byte(r.Intn(256))
			if target(cand) != nil {
				best = cand
				continue
			}
		}
		// If no progress, random deletion
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
