package cache

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type cacheSuite struct {
	suite.Suite
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(cacheSuite))
}

type keyValueTTL struct {
	Key   string
	Value any
	TTL   time.Duration
}

type keyExpireIn struct {
	Key      string
	ExpireIn time.Duration
}

type keyValue struct {
	Key   string
	Value any
}

type state struct {
	Data  []keyValue
	LRULL []string
	TTLH  []string
}

func (s *cacheSuite) TestSet() {
	tt := []struct {
		name         string
		cap          int
		minTickMilli int
		initialData  []keyValueTTL
		addData      keyValueTTL
		wantData     []keyValueTTL
		wantLRUL     []string
		wantTTLH     []keyExpireIn
	}{
		{
			name:         "0. Empty cache",
			cap:          5,
			minTickMilli: 500,
			initialData:  []keyValueTTL{},
			addData: keyValueTTL{
				Key:   "key0",
				Value: "value0",
				TTL:   5 * time.Second,
			},
			wantData: []keyValueTTL{
				{
					Key:   "key0",
					Value: "value0",
					TTL:   5 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key0", "TAIL"},
			wantTTLH: []keyExpireIn{
				{
					Key:      "key0",
					ExpireIn: 5 * time.Second,
				},
			},
		},
		{
			name:         "1. Prefilled cache. Adding more data. Same TTL",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   5 * time.Second,
				},
			},
			addData: keyValueTTL{
				Key:   "key11",
				Value: "value11",
				TTL:   5 * time.Second,
			},
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key11",
					Value: "value11",
					TTL:   5 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key11", "key10", "key01", "key00", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key00", ExpireIn: 5 * time.Second},
				{Key: "key01", ExpireIn: 5 * time.Second},
				{Key: "key10", ExpireIn: 5 * time.Second},
				{Key: "key11", ExpireIn: 5 * time.Second},
			},
		},
		{
			name:         "2. Prefilled cache. Adding more data. Different TTL",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   6 * time.Second,
				},
			},
			addData: keyValueTTL{
				Key:   "key11",
				Value: "value11",
				TTL:   5 * time.Second,
			},
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   6 * time.Second,
				},
				{
					Key:   "key11",
					Value: "value11",
					TTL:   5 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key11", "key10", "key01", "key00", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key11", ExpireIn: 5 * time.Second},
				{Key: "key10", ExpireIn: 6 * time.Second},
				{Key: "key00", ExpireIn: 8 * time.Second},
				{Key: "key01", ExpireIn: 8 * time.Second},
			},
		},
		{
			name:         "3. Same key added. Should move to the head",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   8 * time.Second,
				},
			},
			addData: keyValueTTL{
				Key:   "key00",
				Value: "value02",
				TTL:   8 * time.Second,
			},
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value02",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   8 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key00", "key10", "key01", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key01", ExpireIn: 8 * time.Second},
				{Key: "key10", ExpireIn: 8 * time.Second},
				{Key: "key00", ExpireIn: 8 * time.Second},
			},
		},
		{
			name:         "4. Capacity exceeded",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
				{
					Key:   "key03",
					Value: "value03",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key04",
					Value: "value04",
					TTL:   4 * time.Second,
				},
			},
			addData: keyValueTTL{
				Key:   "key10",
				Value: "value10",
				TTL:   4 * time.Second,
			},
			wantData: []keyValueTTL{
				{
					Key:   "key01",
					Value: "value01",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
				{
					Key:   "key03",
					Value: "value03",
					TTL:   5 * time.Second,
				},
				{
					Key:   "key04",
					Value: "value04",
					TTL:   4 * time.Second,
				},
				{
					Key:   "key10",
					Value: "value10",
					TTL:   4 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key10", "key04", "key03", "key02", "key01", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key10", ExpireIn: 4 * time.Second},
				{Key: "key04", ExpireIn: 4 * time.Second},
				{Key: "key03", ExpireIn: 5 * time.Second},
				{Key: "key02", ExpireIn: 6 * time.Second},
				{Key: "key01", ExpireIn: 8 * time.Second},
			},
		},
	}

	for _, v := range tt {
		s.Run(v.name, func() {
			// 0.0 Creating cache
			cache := NewCacheShard(v.cap, v.minTickMilli, 0)

			// 0.1 Adding initial data
			for _, initialItem := range v.initialData {
				cache.set(initialItem.Key, initialItem.Value, initialItem.TTL)
				time.Sleep(10 * time.Millisecond)
			}

			// 1. Executing Set
			cache.set(v.addData.Key, v.addData.Value, v.addData.TTL)

			// 2.Retreiving cache data
			gotData := make([]keyValueTTL, 0)
			gotTTLH := make([]keyExpireIn, 0)

			LRUNode := cache.LRULL.Head
			gotLRUL := []string{"HEAD"}

			for gotKey, gotValue := range cache.data {
				gotData = append(gotData,
					keyValueTTL{
						Key:   gotKey,
						Value: gotValue.Value,
						TTL:   time.Until(gotValue.TTLElem.ExpireAt).Round(time.Second),
					},
				)
			}

			// 3. Sorting slices
			slices.SortFunc(gotData, func(a, b keyValueTTL) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			slices.SortFunc(v.wantData, func(a, b keyValueTTL) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			// 4. Retreiving LRU linked list data
			for LRUNode.Next.Next != nil {
				LRUNode = LRUNode.Next
				gotLRUL = append(gotLRUL, LRUNode.Key)
			}
			gotLRUL = append(gotLRUL, "TAIL")

			// 5. Retreiving TTL heap data
			for _, TTLNode := range cache.TTLH.Nodes {
				keyExpireInUnit := keyExpireIn{
					Key:      TTLNode.Key,
					ExpireIn: time.Until(TTLNode.ExpireAt).Round(time.Second),
				}
				gotTTLH = append(gotTTLH, keyExpireInUnit)
			}

			// 5.1 Sorting TTL data (the order of elements in the heap is not guaranteed)
			slices.SortFunc(gotTTLH, func(a, b keyExpireIn) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			slices.SortFunc(v.wantTTLH, func(a, b keyExpireIn) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			// 6. Comparing data
			s.Equal(v.wantData, gotData)
			s.Equal(v.wantLRUL, gotLRUL)
			s.Equal(v.wantTTLH, gotTTLH)
		})
	}
}

func (s *cacheSuite) TestGet() {
	tt := []struct {
		name         string
		cap          int
		minTickMilli int
		initialData  []keyValueTTL
		key          string
		wantValue    any
		wantExists   bool
		wantData     []keyValueTTL
		wantLRUL     []string
		wantTTLH     []keyExpireIn
	}{
		{
			name:         "0. Key is present",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   7 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			key:        "key01",
			wantExists: true,
			wantValue:  "value01",
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   7 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key01", "key02", "key00", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key02", ExpireIn: 6 * time.Second},
				{Key: "key01", ExpireIn: 7 * time.Second},
				{Key: "key00", ExpireIn: 8 * time.Second},
			},
		},
		{
			name:         "1. Key is absent",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   7 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			key:        "key10",
			wantExists: false,
			wantValue:  nil,
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   7 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key02", "key01", "key00", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key02", ExpireIn: 6 * time.Second},
				{Key: "key01", ExpireIn: 7 * time.Second},
				{Key: "key00", ExpireIn: 8 * time.Second},
			},
		},
		{
			name:         "2. Key is expired",
			cap:          5,
			minTickMilli: 500,
			initialData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   7 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			key:        "key01",
			wantExists: false,
			wantValue:  nil,
			wantData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   8 * time.Second,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   6 * time.Second,
				},
			},
			wantLRUL: []string{"HEAD", "key02", "key00", "TAIL"},
			wantTTLH: []keyExpireIn{
				{Key: "key02", ExpireIn: 6 * time.Second},
				{Key: "key00", ExpireIn: 8 * time.Second},
			},
		},
	}

	for _, v := range tt {
		s.Run(v.name, func() {
			// 0.0 Creating cache
			cache := NewCacheShard(v.cap, v.minTickMilli, 0)

			// 0.1 Adding initial data
			for _, initialItem := range v.initialData {
				cache.set(initialItem.Key, initialItem.Value, initialItem.TTL)
				time.Sleep(10 * time.Millisecond)
			}

			// 1. Executing Get
			gotValue, gotExists := cache.get(v.key)

			// 2.Retreiving cache data
			gotData := make([]keyValueTTL, 0)
			gotTTLH := make([]keyExpireIn, 0)

			LRUNode := cache.LRULL.Head
			gotLRUL := []string{"HEAD"}

			for gotKey, gotValue := range cache.data {
				gotData = append(gotData,
					keyValueTTL{
						Key:   gotKey,
						Value: gotValue.Value,
						TTL:   time.Until(gotValue.TTLElem.ExpireAt).Round(time.Second),
					},
				)
			}

			// 3. Sorting slices
			slices.SortFunc(gotData, func(a, b keyValueTTL) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			slices.SortFunc(v.wantData, func(a, b keyValueTTL) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			// 4. Retreiving LRU linked list data
			for LRUNode.Next.Next != nil {
				LRUNode = LRUNode.Next
				gotLRUL = append(gotLRUL, LRUNode.Key)
			}
			gotLRUL = append(gotLRUL, "TAIL")

			// 5. Retreiving TTL heap data
			for _, TTLNode := range cache.TTLH.Nodes {
				keyExpireInUnit := keyExpireIn{
					Key:      TTLNode.Key,
					ExpireIn: time.Until(TTLNode.ExpireAt).Round(time.Second),
				}
				gotTTLH = append(gotTTLH, keyExpireInUnit)
			}

			// 5.1 Sorting TTL data (the order of elements in the heap is not guaranteed)
			slices.SortFunc(gotTTLH, func(a, b keyExpireIn) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			slices.SortFunc(v.wantTTLH, func(a, b keyExpireIn) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			// 6. Comparing data
			s.Equal(v.wantExists, gotExists)
			s.Equal(v.wantValue, gotValue)

			s.Equal(v.wantData, gotData)
			s.Equal(v.wantLRUL, gotLRUL)
			s.Equal(v.wantTTLH, gotTTLH)
		})
	}
}

func (s *cacheSuite) TestWork() {
	tt := []struct {
		name         string
		cap          int
		minTickMilli int
		setData      []keyValueTTL
		key          string
		wantValue    any
		wantExists   bool
		sleepTime    time.Duration
		wantState    map[string]state
	}{
		{
			name:         "0. One expired",
			cap:          10,
			minTickMilli: 500,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   540 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   530 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   490 * time.Millisecond,
				},
			},
			key:        "key01",
			wantValue:  "value01",
			wantExists: true,
			sleepTime:  491 * time.Millisecond,
			wantState: map[string]state{
				"after_sleep": {
					Data: []keyValue{
						{
							Key:   "key00",
							Value: "value00",
						},
						{
							Key:   "key01",
							Value: "value01",
						},
					},
					LRULL: []string{"HEAD", "key01", "key00", "TAIL"},
					TTLH:  []string{"key01", "key00"},
				},
			},
		},
		{
			name:         "1. All expired",
			cap:          10,
			minTickMilli: 500,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   410 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   420 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   430 * time.Millisecond,
				},
			},
			key:        "key01",
			wantValue:  nil,
			wantExists: false,
			sleepTime:  495 * time.Millisecond,
			wantState: map[string]state{
				"after_sleep": {
					Data:  []keyValue{},
					LRULL: []string{"HEAD", "TAIL"},
					TTLH:  []string{},
				},
			},
		},
		{
			name:         "2. None expired",
			cap:          10,
			minTickMilli: 500,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   560 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   540 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   520 * time.Millisecond,
				},
			},
			key:        "key01",
			wantValue:  "value01",
			wantExists: true,
			sleepTime:  500 * time.Millisecond,
			wantState: map[string]state{
				"after_sleep": {
					Data: []keyValue{
						{
							Key:   "key00",
							Value: "value00",
						},
						{
							Key:   "key01",
							Value: "value01",
						},
						{
							Key:   "key02",
							Value: "value02",
						},
					},
					LRULL: []string{"HEAD", "key01", "key02", "key00", "TAIL"},
					TTLH:  []string{"key02", "key00", "key01"},
				},
			},
		},
	}

	for _, v := range tt {
		s.Run(v.name, func() {
			// 0.0 Creating cache
			cache := NewCacheShard(v.cap, v.minTickMilli, 0)

			// 1. Set data
			for _, setItem := range v.setData {
				cache.set(setItem.Key, setItem.Value, setItem.TTL)
				time.Sleep(10 * time.Millisecond)
			}

			// 2. Sleep
			time.Sleep(v.sleepTime)

			// 3. Get data
			gotValue, gotExists := cache.get(v.key)

			// 4. Freeze
			cache.freeze()

			// 5. Comparing data
			s.Equal(v.wantValue, gotValue)
			s.Equal(v.wantExists, gotExists)

			// 6.1 Checking state after sleep
			gotData, gotLRUL, gotTTLH := cache.getState()

			// 6.2 Checking status after sleep
			s.Equal(v.wantState["after_sleep"].LRULL, gotLRUL)
			s.Equal(v.wantState["after_sleep"].TTLH, gotTTLH)

			// 6.3 Sorting wantData slice
			wantData := v.wantState["after_sleep"].Data

			slices.SortFunc(wantData, func(a, b keyValue) int {
				if a.Key < b.Key {
					return -1
				}
				if a.Key > b.Key {
					return 1
				}
				return 0
			})

			// 7. Comparing data
			s.Equal(wantData, gotData)
		})
	}
}

func (s *cacheSuite) TestSharded() {
	tt := []struct {
		name         string
		cap          int
		shardsNum    int
		minTickMilli int
		waitDuration time.Duration
		setData      []keyValueTTL
		wantData     []keyValue
	}{
		{
			name:         "0. All found",
			cap:          500,
			shardsNum:    16,
			minTickMilli: 500,
			waitDuration: time.Millisecond * 400,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   560 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   540 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   520 * time.Millisecond,
				},
				{
					Key:   "alice",
					Value: "azaza",
					TTL:   580 * time.Millisecond,
				},
				{
					Key:   "bob",
					Value: "bzbzb",
					TTL:   580 * time.Millisecond,
				},
			},
			wantData: []keyValue{
				{
					Key:   "key00",
					Value: "value00",
				},
				{
					Key:   "key01",
					Value: "value01",
				},
				{
					Key:   "key02",
					Value: "value02",
				},
				{
					Key:   "alice",
					Value: "azaza",
				},
				{
					Key:   "bob",
					Value: "bzbzb",
				},
			},
		},
		{
			name:         "1. Some found. One expired",
			cap:          500,
			shardsNum:    16,
			minTickMilli: 500,
			waitDuration: time.Millisecond * 510,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: "value00",
					TTL:   560 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: "value01",
					TTL:   540 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   450 * time.Millisecond,
				},
			},
			wantData: []keyValue{
				{
					Key:   "key00",
					Value: "value00",
				},
				{
					Key:   "key01",
					Value: "value01",
				},
			},
		},
		{
			name:         "2. Some found. One expired. Different value types",
			cap:          500,
			shardsNum:    16,
			minTickMilli: 500,
			waitDuration: time.Millisecond * 510,
			setData: []keyValueTTL{
				{
					Key:   "key00",
					Value: 0,
					TTL:   560 * time.Millisecond,
				},
				{
					Key:   "key01",
					Value: struct{}{},
					TTL:   540 * time.Millisecond,
				},
				{
					Key:   "key02",
					Value: "value02",
					TTL:   450 * time.Millisecond,
				},
			},
			wantData: []keyValue{
				{
					Key:   "key00",
					Value: 0,
				},
				{
					Key:   "key01",
					Value: struct{}{},
				},
			},
		},
	}

	for _, v := range tt {
		s.Run(v.name, func() {

			shardedCache := NewCacheSharded(v.cap, v.shardsNum, v.minTickMilli)

			for _, dataPiece := range v.setData {
				shardedCache.Set(dataPiece.Key, dataPiece.Value, dataPiece.TTL)
			}

			time.Sleep(v.waitDuration)

			shardedCache.Freeze()

			gotData := make([]keyValue, 0, len(v.setData))

			for _, setDataPiece := range v.setData {
				if value, exists := shardedCache.Get(setDataPiece.Key); exists {
					gotData = append(gotData, keyValue{
						Key:   setDataPiece.Key,
						Value: value,
					})
				}

			}

			s.Equal(v.wantData, gotData)
		})
	}
}

func (c *TTLLRUCacheShard) getState() ([]keyValue, []string, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	gotLRUL := []string{"HEAD"}
	gotTTLH := make([]string, 0)
	gotData := make([]keyValue, 0)
	LRUNode := c.LRULL.Head

	// 1. LRU linked list
	for LRUNode.Next.Next != nil {
		LRUNode = LRUNode.Next
		gotLRUL = append(gotLRUL, LRUNode.Key)
	}
	gotLRUL = append(gotLRUL, "TAIL")

	// 2. TTL heap
	for _, TTLNode := range c.TTLH.Nodes {
		gotTTLH = append(gotTTLH, TTLNode.Key)
	}

	// 3.0 Key-value-TTL slice
	for gotKey, gotValue := range c.data {
		gotData = append(gotData,
			keyValue{
				Key:   gotKey,
				Value: gotValue.Value,
			},
		)
	}

	// 3.1 Sorting slice
	slices.SortFunc(gotData, func(a, b keyValue) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		return 0
	})

	return gotData, gotLRUL, gotTTLH
}
