package parallelize

import (
	"context"
	"k8s.io/client-go/util/workqueue"
	"math"
)

const (
	// DefaultParallelism is the default level of parallelism used for scheduling workloads.
	DefaultParallelism = 16
)

// Parallelizer holds the parallelism for scheduler.
type Parallelizer struct {
	parallelism int
}

// NewParallelizer returns an object holding the parallelism.
func NewParallelizer(p int) *Parallelizer {
	if p <= 0 {
		p = DefaultParallelism
	}

	return &Parallelizer{parallelism: p}
}

// chunkSize returns chunksize per parallelism unit of work.
func (p *Parallelizer) chunkSize(pieces int) int {
	size := int(math.Sqrt(float64(pieces)))

	if r := pieces/p.parallelism + 1; r < size {
		size = r
	} else if size < 1 {
		size = 1
	}

	return size
}

// Until parallelizes and waits until doWorkPiece is executed for all pieces.
func (p *Parallelizer) Until(ctx context.Context,
	pieces int,
	doWorkPiece workqueue.DoWorkPieceFunc) {
	workqueue.ParallelizeUntil(ctx, p.parallelism,
		pieces, doWorkPiece,
		workqueue.WithChunkSize(p.chunkSize(pieces)))
}
