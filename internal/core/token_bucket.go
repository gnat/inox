package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TOKEN_BUCKET_MANAGEMENT_TICK_INTERVAL = time.Millisecond
	TOKEN_BUCKET_CAPACITY_SCALE           = int64(time.Second / TOKEN_BUCKET_MANAGEMENT_TICK_INTERVAL)

	MAX_WAIT_CHAN_COUNT = 1000
)

func init() {
	startTokenBucketManagerGoroutine()
}

// A tokenBucket represents a thread-safe token bucket, a single goroutine manages the buckets.
type tokenBucket struct {
	lastDecrementTime time.Time

	tokenLock            *sync.Mutex
	capacity             ScaledTokenCount
	available            ScaledTokenCount
	increment            ScaledTokenCount
	pausedDecrementation atomic.Bool
	decrementFn          func(lastDecrementTime time.Time) int64
	context              *Context

	chanListLock    sync.Mutex
	waitChans       []chan (struct{})
	neededTokenList []ScaledTokenCount

	cancelContextOnNegativeCount bool
}

type ScaledTokenCount int64

func (c ScaledTokenCount) RealCount() int64 {
	return int64(c) / TOKEN_BUCKET_CAPACITY_SCALE
}

type tokenBucketConfig struct {
	cap                          int64
	initialAvail                 int64
	fillRate                     int64
	decrementFn                  func(time.Time) int64
	cancelContextOnNegativeCount bool
}

// newBucket returns a new token bucket with specified fillrate & capacity, the bucket is created full.
func newBucket(config tokenBucketConfig) *tokenBucket {
	if config.cap < 0 {
		panic(fmt.Sprintf("token bucket: capacity %v should be > 0", config.cap))
	}

	avail := config.initialAvail
	if avail < 0 {
		avail = config.cap
	}

	tb := &tokenBucket{
		tokenLock:                    &sync.Mutex{},
		capacity:                     ScaledTokenCount(config.cap * TOKEN_BUCKET_CAPACITY_SCALE),
		available:                    ScaledTokenCount(avail * TOKEN_BUCKET_CAPACITY_SCALE),
		increment:                    ScaledTokenCount(config.fillRate),
		decrementFn:                  config.decrementFn,
		cancelContextOnNegativeCount: config.cancelContextOnNegativeCount,
		lastDecrementTime:            time.Now(),
	}

	tokenBucketsLock.Lock()
	tokenBuckets[tb] = struct{}{}
	tokenBucketsLock.Unlock()

	return tb
}

func (tb *tokenBucket) SetContext(ctx *Context) {
	tb.tokenLock.Lock()
	defer tb.tokenLock.Unlock()

	tb.context = ctx
}

func (tb *tokenBucket) Capacity() int64 {
	return tb.capacity.RealCount()
}

func (tb *tokenBucket) Available() int64 {
	tb.tokenLock.Lock()
	defer tb.tokenLock.Unlock()

	return tb.available.RealCount()
}

// TryTake trys to task specified count tokens from the bucket. if there are
// not enough tokens in the bucket, it will return false.
func (tb *tokenBucket) TryTake(count int64) bool {
	scaledCount := ScaledTokenCount(count * TOKEN_BUCKET_CAPACITY_SCALE)
	return tb.tryTake(scaledCount, scaledCount)
}

// Take tasks specified count tokens from the bucket, if there are
// not enough tokens in the bucket, it will keep waiting until count tokens are
// available and then take them.
func (tb *tokenBucket) Take(count int64) {
	tb.waitAndTake(count, count)
}

func (tb *tokenBucket) GiveBack(count int64) {
	tb.tokenLock.Lock()
	defer tb.tokenLock.Unlock()

	tb.available += ScaledTokenCount(count * TOKEN_BUCKET_CAPACITY_SCALE)
	tb.available = min(tb.capacity, tb.available)
}

func (tb *tokenBucket) PauseDecrementation() {
	tb.pausedDecrementation.Store(true)
}

func (tb *tokenBucket) ResumeDecrementation() {
	tb.pausedDecrementation.Store(false)
}

// TakeMaxDuration tasks specified count tokens from the bucket, if there are
// not enough tokens in the bucket, it will keep waiting until count tokens are
// available and then take them or just return false when reach the given max
// duration.
func (tb *tokenBucket) TakeMaxDuration(count int64, max time.Duration) bool {
	return tb.waitAndTakeMaxDuration(count, count, max)
}

// Wait will keep waiting until count tokens are available in the bucket.
func (tb *tokenBucket) Wait(count int64) {
	tb.waitAndTake(count, 0)
}

// WaitMaxDuration will keep waiting until count tokens are available in the
// bucket or just return false when reach the given max duration.
func (tb *tokenBucket) WaitMaxDuration(count int64, max time.Duration) bool {
	return tb.waitAndTakeMaxDuration(count, 0, max)
}

func (tb *tokenBucket) tryTake(need, use ScaledTokenCount) bool {
	tb.checkCount(need)

	tb.tokenLock.Lock()
	defer tb.tokenLock.Unlock()

	if need <= tb.available {
		tb.available -= use

		return true
	}

	return false
}

func (tb *tokenBucket) addWaitChannel(need ScaledTokenCount) chan (struct{}) {
	var channel chan (struct{})
	if len(waitChanPool) == 0 {
		channel = make(chan struct{}, 1)
	} else {
		channel = <-waitChanPool
	}
	tb.chanListLock.Lock()
	tb.waitChans = append(tb.waitChans, channel)
	tb.neededTokenList = append(tb.neededTokenList, need)
	tb.chanListLock.Unlock()
	return channel
}

func (tb *tokenBucket) waitAndTake(need, use int64) {
	needCount := ScaledTokenCount(need * TOKEN_BUCKET_CAPACITY_SCALE)
	useCount := ScaledTokenCount(use * TOKEN_BUCKET_CAPACITY_SCALE)

	if ok := tb.tryTake(needCount, useCount); ok {
		return
	}

	waitChan := tb.addWaitChannel(needCount)
	<-waitChan
}

func (tb *tokenBucket) waitAndTakeMaxDuration(need, use int64, max time.Duration) bool {
	needCount := ScaledTokenCount(need * TOKEN_BUCKET_CAPACITY_SCALE)
	useCount := ScaledTokenCount(use * TOKEN_BUCKET_CAPACITY_SCALE)

	if ok := tb.tryTake(needCount, useCount); ok {
		return true
	}

	waitChan := tb.addWaitChannel(needCount)

	select {
	case <-waitChan:
		return true
	case <-time.After(max):
		return false
	}
}

func (tb *tokenBucket) Destroy() {
	tokenBucketsLock.Lock()
	defer tokenBucketsLock.Unlock()
	delete(tokenBuckets, tb)
}

func (tb *tokenBucket) checkCount(count ScaledTokenCount) {
	if count < 0 || count > tb.capacity {
		panic(fmt.Sprintf("token-bucket: count %v should be less than bucket's"+
			" capacity %v", count, tb.capacity))
	}
}

var (
	tokenBucketManagerStarted atomic.Bool
	tokenBuckets              = map[*tokenBucket]struct{}{}
	tokenBucketsLock          sync.Mutex

	waitChanPool = make(chan (chan (struct{})), MAX_WAIT_CHAN_COUNT)
)

func startTokenBucketManagerGoroutine() {
	if !tokenBucketManagerStarted.CompareAndSwap(false, true) {
		return
	}

	updateTokenCount := func(tb *tokenBucket) {
		tb.tokenLock.Lock()
		defer tb.tokenLock.Unlock()

		if tb.decrementFn == nil {
			if tb.available < tb.capacity {
				increment := tb.increment
				tb.available = tb.available + increment
			}
		} else if !tb.pausedDecrementation.Load() {
			tb.available -= ScaledTokenCount(tb.decrementFn(tb.lastDecrementTime) * TOKEN_BUCKET_CAPACITY_SCALE)
		}

		if tb.available < 0 && tb.cancelContextOnNegativeCount && tb.context != nil {
			tb.context.Cancel() // add reason
			return
		}

		tb.available = max(0, tb.available)
		tb.lastDecrementTime = time.Now()

		func() {
			tb.chanListLock.Lock()
			defer tb.chanListLock.Unlock()

			for len(tb.waitChans) >= 1 { // if at least one goroutine is waiting for the bucket to refill
				waitChan := tb.waitChans[len(tb.waitChans)-1]
				neededCount := tb.neededTokenList[len(tb.waitChans)-1]

				if tb.available >= neededCount {
					newLength := len(tb.waitChans) - 1
					tb.waitChans = tb.waitChans[:newLength]
					tb.neededTokenList = tb.neededTokenList[:newLength]

					tb.available -= neededCount
					waitChan <- struct{}{} //resume the waiting goroutine
					if len(waitChanPool) < cap(waitChanPool) {
						waitChanPool <- waitChan
					}
				} else {
					break
				}
			}
		}()
	}

	go func() {
		ticks := time.Tick(TOKEN_BUCKET_MANAGEMENT_TICK_INTERVAL)

		for range ticks {
			func() {
				tokenBucketsLock.Lock()
				defer recover()
				defer tokenBucketsLock.Unlock()

				for bucket := range tokenBuckets {
					updateTokenCount(bucket)
				}
			}()
		}

	}()
}
