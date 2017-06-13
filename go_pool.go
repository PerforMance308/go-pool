package go_pool

import (
	"server/tree/rbtree"
	"sync"
	"reflect"
	"time"
	"github.com/pkg/errors"
)

const defaultTimeout = 5

type PoolArgs struct {
	Size     int
	TimeOut  int64
}

type StartPool func() (interface{}, error)

type pools struct {
	PoolArgs
	pool     []*poolInfo
	use      *rbtree.Tree
	idle     []*poolInfo
	mu       sync.RWMutex
	initFunc StartPool
}

type poolInfo struct {
	id   interface{}
	pool interface{}
}

var _pools *pools

func GetPools() *pools {
	return _pools
}

func Start(poolArgs PoolArgs, startFunc StartPool) error {
	pools := &pools{}

	var mu sync.RWMutex

	pools.mu = mu
	pools.PoolArgs = poolArgs
	pools.use = rbtree.NewWithIntComparator()
	pools.idle = make([]*poolInfo, 0)
	pools.pool = make([]*poolInfo, 0)
	pools.initFunc = startFunc
	if pools.TimeOut == 0 {
		pools.TimeOut = defaultTimeout
	}

	for i := 0; i < pools.Size; i++ {
		if err := pools.newPool(); err != nil {
			return err
		}
	}

	_pools = pools
	return nil
}

func Transaction(funcName string, args interface{}) (interface{}, error) {
	return GetPools().transaction(funcName, args)
}

func (pools *pools) transaction(funcName string, args interface{}) (interface{}, error) {
	pool := pools.CheckOut()
	defer pools.CheckIn(pool)

	method, found := reflect.TypeOf(pool.pool).MethodByName(funcName)
	if !found {
		return nil, errors.New("cant find method")
	}

	res := method.Func.Call([]reflect.Value{reflect.ValueOf(pool.pool), reflect.ValueOf(args).Elem()})

	if len(res) == 1 {
		err := res[0].Interface()
		if err != nil {
			return nil, err.(error)
		}
		return nil, nil
	} else if len(res) == 2 {
		err := res[1].Interface()
		if err != nil {
			return nil, err.(error)
		}

		return res[0].Interface(), nil
	}

	return nil, errors.New("transaction too many return value")
}

func (pools *pools) CheckOut() poolInfo {
	ch := make(chan poolInfo)
	timeOutChan := make(chan bool)

	go pools.checkout(ch, timeOutChan)

	select {
	case poolInfo := <-ch:
		return poolInfo
	case <-timeOutChan:
		return poolInfo{}
	}
}

func (pools *pools) checkout(ch chan poolInfo, timeOutChan chan bool) {
	pools.mu.Lock()
	if len(pools.idle) > 0 {
		pool := pools.idle[0]
		pools.idle = pools.idle[1:]
		pools.use.Put(pool.id, pool.pool)

		ch <- *pool
		pools.mu.Unlock()
	} else {
		if len(pools.pool) < pools.Size {
			pools.newPool()
			pool := pools.idle[0]
			pools.idle = pools.idle[1:]
			pools.use.Put(pool.id, pool.pool)

			ch <- *pool
			pools.mu.Unlock()
		} else {
			pools.mu.Unlock()
			now := time.Now().UTC().Unix()
			for {
				if (time.Now().UTC().Unix() - now) > pools.TimeOut {
					timeOutChan <- true
					break
				} else {
					pools.mu.Lock()
					if len(pools.idle) > 0 {
						pool := pools.idle[0]
						pools.idle = pools.idle[1:]
						pools.use.Put(pool.id, pool.pool)

						ch <- *pool
						pools.mu.Unlock()
						break
					} else {
						pools.mu.Unlock()
						continue
					}
				}
			}
		}
	}

	close(ch)
	close(timeOutChan)
}

func (pools *pools) CheckIn(pInfo poolInfo) {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	pools.use.Remove(pInfo.id)
	pools.idle = append(pools.idle, &pInfo)
}

func (pools *pools) newPool() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	pool, err := pools.initFunc()
	if err != nil {
		return err
	}

	poolId := len(pools.pool) + 1
	pInfo := &poolInfo{id: poolId, pool: pool}
	pools.pool = append(pools.pool, pInfo)
	pools.idle = append(pools.idle, pInfo)

	return nil
}