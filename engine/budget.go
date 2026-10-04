package main

import "sync/atomic"

type byteBudget struct {
	limit uint64
	used  atomic.Uint64
}

func newByteBudget(limit uint64) *byteBudget { return &byteBudget{limit: limit} }

func (b *byteBudget) Reserve(n uint64) uint64 {
	for {
		used := b.used.Load()
		if used >= b.limit {
			return 0
		}
		allowed := n
		if remaining := b.limit - used; allowed > remaining {
			allowed = remaining
		}
		if b.used.CompareAndSwap(used, used+allowed) {
			return allowed
		}
	}
}

func (b *byteBudget) Release(n uint64) {
	for {
		used := b.used.Load()
		if n > used {
			n = used
		}
		if b.used.CompareAndSwap(used, used-n) {
			return
		}
	}
}
