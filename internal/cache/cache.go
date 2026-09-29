package cache

import (
	"math/bits"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/cache/main/internal/lru"
	"github.com/cache/main/internal/ttl"
)

type TTLLRUCache interface {
	Set(key string, value any, ttl time.Duration)
	Get(key string) (value any, exists bool)

	// Freeze stops the background eviction worker to conserve CPU resources.
	// The cache remains fully accessible for concurrent reads (Get) and writes (Set).
	// After calling Freeze, expired items are evicted lazily upon calling Get.
	Freeze()
}

type CacheNode struct {
	Value   any
	LRUElem *lru.Node
	TTLElem *ttl.Node
}

func newCacheNode(key string, value any, expireAt time.Time, TTLHeapIndex int) *CacheNode {

	return &CacheNode{
		Value: value,
		LRUElem: &lru.Node{
			Key: key,
		},
		TTLElem: &ttl.Node{
			Key:       key,
			ExpireAt:  expireAt,
			HeapIndex: TTLHeapIndex,
		},
	}
}

type TTLLRUCacheShard struct {
	id      uint64
	mu      sync.Mutex
	data    map[string]*CacheNode
	cap     int
	minTick time.Duration
	LRULL   lru.LinkedList
	TTLH    ttl.Heap

	doneChan       chan struct{}
	freezeChan     chan struct{}
	resetTimerChan chan struct{}
}

type ShardedCache struct {
	shards    []*TTLLRUCacheShard
	shardMask uint64
}

func NewCacheShard(cap int, minTickMilli int, id int) *TTLLRUCacheShard {
	m := make(map[string]*CacheNode, cap)
	ttlHeap := ttl.NewHeap(cap)
	lruLL := lru.NewLRULinkedList()

	doneChan := make(chan struct{}, 1)
	freezeChan := make(chan struct{})
	reserTimerChan := make(chan struct{}, 1)

	cache := &TTLLRUCacheShard{
		id:      uint64(id),
		mu:      sync.Mutex{},
		data:    m,
		minTick: time.Duration(minTickMilli) * time.Millisecond,
		cap:     cap,
		LRULL:   lruLL,
		TTLH:    ttlHeap,

		doneChan:       doneChan,
		freezeChan:     freezeChan,
		resetTimerChan: reserTimerChan,
	}

	go cache.startCleanupWorker()

	return cache
}

func NewCacheSharded(totalCap int, requestedShards int, minTickMilli int) *ShardedCache {
	if requestedShards <= 0 {
		requestedShards = 32
	}

	shardCount := max(1<<bits.Len(uint(requestedShards-1)), 2)   // shardCount =...= 2^(bits in (requestedShards-1)), at least 2
	shardCap := max(((totalCap+shardCount-1)/shardCount)*3/2, 1) // shardCap =...= 150%(totalCap/ shardCount), at least 1

	shards := make([]*TTLLRUCacheShard, shardCount)

	for i := range shards {
		shards[i] = NewCacheShard(shardCap, minTickMilli, i)
	}

	return &ShardedCache{
		shards:    shards,
		shardMask: uint64(shardCount - 1),
	}
}

func (c *TTLLRUCacheShard) set(key string, value any, ttl time.Duration) {
	if len(key) == 0 || value == nil {

		return
	}

	if ttl == 0 {
		ttl = c.minTick
	}

	c.mu.Lock()

	if oldCacheNode, ok := c.data[key]; ok {
		oldCacheNode.Value = value
		oldCacheNode.TTLElem.ExpireAt = time.Now().Add(ttl)

		c.TTLH.ShiftUp(oldCacheNode.TTLElem.HeapIndex)
		c.TTLH.ShiftDown(oldCacheNode.TTLElem.HeapIndex)
		c.LRULL.MoveToHead(oldCacheNode.LRUElem)

		if oldCacheNode.TTLElem.HeapIndex == 0 {

			select {
			case c.resetTimerChan <- struct{}{}:
			default:
				// If the channel buffer is full, drop the signal and proceed.
				// The worker is guaranteed to wake up since a signal is already pending.
			}
		}

		c.mu.Unlock()

		return
	}

	// Not found

	if len(c.data) == c.cap {
		keyToDelete := c.LRULL.RemoveTail()
		heapIndexToRemove := c.data[keyToDelete].TTLElem.HeapIndex
		c.TTLH.Remove(heapIndexToRemove)

		delete(c.data, keyToDelete)
	}

	newCacheNodeExample := newCacheNode(key, value, time.Now().Add(ttl), len(c.TTLH.Nodes))

	c.data[key] = newCacheNodeExample
	c.TTLH.Nodes = append(c.TTLH.Nodes, newCacheNodeExample.TTLElem)
	c.LRULL.InsertAtHead(newCacheNodeExample.LRUElem)

	heapCurrentSize := len(c.TTLH.Nodes)
	c.TTLH.ShiftUp(heapCurrentSize - 1)

	if newCacheNodeExample.TTLElem.HeapIndex == 0 {

		select {
		case c.resetTimerChan <- struct{}{}:
		default:
			// If the channel buffer is full, drop the signal and proceed.
			// The worker is guaranteed to wake up since a signal is already pending.
		}
	}

	c.mu.Unlock()
}

func (s *ShardedCache) Set(key string, value any, ttl time.Duration) {
	shard := s.getShard(key)

	shard.set(key, value, ttl)
}

func (c *TTLLRUCacheShard) get(key string) (value any, exists bool) {
	if len(key) == 0 {

		return nil, false
	}

	c.mu.Lock()

	if foundCacheNode, ok := c.data[key]; ok {

		if foundCacheNode.TTLElem.ExpireAt.Before(time.Now()) {
			c.LRULL.Remove(foundCacheNode.LRUElem)
			heapIndexToRemove := foundCacheNode.TTLElem.HeapIndex
			c.TTLH.Remove(heapIndexToRemove)

			delete(c.data, key)
			c.mu.Unlock()

			return nil, false
		}

		c.LRULL.MoveToHead(foundCacheNode.LRUElem)
		c.mu.Unlock()

		return foundCacheNode.Value, true
	}

	// not found

	return nil, false
}

func (s *ShardedCache) Get(key string) (value any, exists bool) {
	shard := s.getShard(key)

	return shard.get(key)
}

func (c *TTLLRUCacheShard) freeze() {
	c.mu.Lock()

	defer func() {
		c.mu.Unlock()
		<-c.doneChan
	}()

	select {
	case <-c.freezeChan:

		return
	default:
	}

	close(c.freezeChan)
}

func (s *ShardedCache) Freeze() {
	for _, shard := range s.shards {
		shard.freeze()
	}
}

func (c *TTLLRUCacheShard) removeCacheExpired() (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiredKeys := c.TTLH.RemoveHeapExpired(time.Now())

	for _, key := range expiredKeys {
		if cacheNode, ok := c.data[key]; ok {
			c.LRULL.Remove(cacheNode.LRUElem)
			delete(c.data, key)
		}
	}

	if len(c.TTLH.Nodes) == 0 {

		return c.minTick, false
	}

	actualTimeLeft := max(time.Until(c.TTLH.Nodes[0].ExpireAt), c.minTick)
	maxSleepDuration := c.minTick * 10
	timeLeft := max(c.minTick, min(actualTimeLeft, maxSleepDuration)) // Protection: if actualTimeLeft > 10*minTick,  timeLeft = 10*minTick

	return timeLeft, true
}

func (c *TTLLRUCacheShard) startCleanupWorker() {
	defer close(c.doneChan)

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	for {
		timeLeft, hasElements := c.removeCacheExpired()

		if hasElements {
			timer.Reset(timeLeft)
		} else {
			timer.Stop()
		}

		select {
		case <-c.freezeChan:
			return

		case <-timer.C:

		case <-c.resetTimerChan:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
	}
}

func (s *ShardedCache) getShard(key string) *TTLLRUCacheShard {
	idx := xxhash.Sum64String(key) & s.shardMask // idx =...= hash_number * bit_mask (i.e 10001 & 00011 = 00001(2x) => 1(10x))

	return s.shards[idx]
}
