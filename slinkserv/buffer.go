package slinkserv

import (
	"sync"
	"sync/atomic"
)

// TODO: Add an 'alive' property so that a 'close' call stops the listeners.

type BytePipe struct {
	buf      []byte     // Internal buffer
	cap      uint32     // capacity of the buffer
	s        uint32     // s==state 1==alive 0==closed
	padding2 []byte     // used to increase performance... lol
	r        uint32     // Read index
	rc       *sync.Cond // ReadCondition -- used when reader needs to
	padding  []byte     // used to increase performance... lol
	w        uint32     // Write index
	wc       *sync.Cond // WriteCondition -- used when writer has no more room
}

func NewBytePipe(cap uint32) *BytePipe {
	if cap == 0 {
		cap = 32768 // 32kb of room for buffering.
	}
	return &BytePipe{
		buf:      make([]byte, cap),
		s:        1,
		r:        0,
		w:        0,
		rc:       sync.NewCond(&sync.Mutex{}),
		wc:       sync.NewCond(&sync.Mutex{}),
		cap:      cap,
		padding:  make([]byte, 64),
		padding2: make([]byte, 64),
	}
}

func (bp *BytePipe) Len() int {
	r := atomic.LoadUint32(&bp.r)
	w := atomic.LoadUint32(&bp.w)
	if r <= w {
		return int(w - r)
	}

	return int((bp.cap - r) + (r - w))
}

func (bp *BytePipe) Write(b []byte) int {
	if atomic.LoadUint32(&bp.s) == 0 {
		return 0
	}
	r := atomic.LoadUint32(&bp.r)
	w := atomic.LoadUint32(&bp.w)
	l := uint32(len(b))

	if (w >= r && w+l <= bp.cap) || (r > w && r-w > l) {
		// more than enough space to write entire thing into it.
		copy(bp.buf[w:], b)
		atomic.StoreUint32(&bp.w, w+l)
		bp.rc.Signal()
		return len(b)
	}
	if (w == bp.cap && r == 0) || w == r-1 {
		// no space eft until reader reads something
		bp.wc.L.Lock()
		if r == atomic.LoadUint32(&bp.r) {
			bp.wc.Wait()
		}
		bp.wc.L.Unlock()
		return bp.Write(b)
	}
	if w < r {
		// not enough space! write what we can now, and then call write again to wait.
		bidx := r - w - 1 // cant overwrite that last byte!
		copy(bp.buf[w:], b[:bidx])
		atomic.StoreUint32(&bp.w, r-1)
		bp.rc.Signal()
		return bp.Write(b[bidx:]) // Now write the rest
	}
	// this means we dont have enought space at the end, write and try again.
	bidx := bp.cap - w
	copy(bp.buf[w:], b[:bidx])
	if r == 0 {
		atomic.StoreUint32(&bp.w, bp.cap)
	} else {
		atomic.StoreUint32(&bp.w, 0)
	}
	bp.rc.Signal()
	return bp.Write(b[bidx:]) // Now write the rest
}

func (bp *BytePipe) Read(b []byte) int {
	if atomic.LoadUint32(&bp.s) == 0 {
		return 0
	}

	r := atomic.LoadUint32(&bp.r)
	w := atomic.LoadUint32(&bp.w)
	l := uint32(len(b))

	if (r > w && r+l < bp.cap) || (r < w && w-r > l) {
		// Have more bytes ready than len(b), just do it
		copy(b, bp.buf[r:r+l])
		atomic.StoreUint32(&bp.r, r+l)
		bp.wc.Signal()
		return int(l)
	}
	if r > w {
		// we have more bytes, but need to wrap around to finish reading.
		copy(b, bp.buf[r:bp.cap])
		atomic.StoreUint32(&bp.r, 0)
		bp.wc.Signal()
		return bp.Read(b[bp.cap-r:]) + int(bp.cap-r)
	}
	if r == w {
		// We dont have anything to read, wait for now
		bp.rc.L.Lock()
		if w == atomic.LoadUint32(&bp.w) {
			bp.rc.Wait()
		}
		bp.rc.L.Unlock()
		return bp.Read(b)
	}
	// Finally, this means we have less bytes available than size of b, copy what we have.
	copy(b, bp.buf[r:w])
	atomic.StoreUint32(&bp.r, w)
	bp.wc.Signal()
	return int(w - r)
}

func (bp *BytePipe) Close() {
	atomic.StoreUint32(&bp.s, 0)

	bp.wc.L.Lock()
	bp.wc.Signal()
	bp.wc.L.Unlock()

	bp.rc.L.Lock()
	bp.rc.Signal()
	bp.rc.L.Unlock()
}
