/*
Copyright 2022 Ciena Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parallelize

import (
	"context"
	"math"

	"k8s.io/client-go/util/workqueue"
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
