package snowflake

import (
	"sync"
	"time"
)

const (
	epoch          = int64(1609459200000) // 2021-01-01 00:00:00 UTC
	workerBits     = uint(10)
	sequenceBits   = uint(12)
	workerShift    = sequenceBits
	timestampShift = sequenceBits + workerBits
	maxWorker      = int64(-1) ^ (int64(-1) << workerBits)
	maxSequence    = int64(-1) ^ (int64(-1) << sequenceBits)
)

type Generator struct {
	mu       sync.Mutex
	workerID int64
	sequence int64
	lastTime int64
}

func NewGenerator(workerID int64) (*Generator, error) {
	if workerID < 0 || workerID > maxWorker {
		return nil, &InvalidWorkerIDError{WorkerID: workerID}
	}
	return &Generator{
		workerID: workerID,
		lastTime: -1,
		sequence: 0,
	}, nil
}

func (g *Generator) NextID() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < g.lastTime {
		// Clock moved backwards — we'll wait
		time.Sleep(time.Duration(g.lastTime-now) * time.Millisecond)
		now = time.Now().UnixMilli()
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			// Sequence overflow, wait for next millisecond
			for now <= g.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	return uint64(
		((now - epoch) << timestampShift) |
			(g.workerID << workerShift) |
			g.sequence,
	)
}

type InvalidWorkerIDError struct {
	WorkerID int64
}

func (e *InvalidWorkerIDError) Error() string {
	return "snowflake: invalid worker ID"
}
