package generator

import (
	"encoding/binary"
	"hash/fnv"
	"math/rand"
)

// RNGFactory creates deterministic random generators for workers and rows.
type RNGFactory struct {
	masterSeed int64
}

func NewRNGFactory(masterSeed int64) RNGFactory {
	return RNGFactory{masterSeed: masterSeed}
}

// WorkerRNG returns a deterministic RNG scoped to a worker id.
func (f RNGFactory) WorkerRNG(workerID int) *rand.Rand {
	seed := mixSeed(f.masterSeed, int64(workerID), 0)
	return rand.New(rand.NewSource(seed))
}

// RowRNG returns a deterministic RNG scoped to a logical row index.
// This keeps results reproducible independently from goroutine scheduling.
func (f RNGFactory) RowRNG(rowIndex int) *rand.Rand {
	seed := mixSeed(f.masterSeed, int64(rowIndex), 1)
	return rand.New(rand.NewSource(seed))
}

func mixSeed(master, x, domain int64) int64 {
	h := fnv.New64a()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(master))
	_, _ = h.Write(b[:])
	binary.LittleEndian.PutUint64(b[:], uint64(x))
	_, _ = h.Write(b[:])
	binary.LittleEndian.PutUint64(b[:], uint64(domain))
	_, _ = h.Write(b[:])
	return int64(h.Sum64() & 0x7fffffffffffffff)
}
