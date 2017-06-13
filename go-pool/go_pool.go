package go_pool

import (
	"sync"
	"reflect"
	"time"
	"errors"
	"container/list"
)

const defaultTimeout = 20

type PoolArgs struct {
	Size     int
	TimeOut  int64
}

type StartPool func() (interface{}, error)

type pools struct {
	PoolArgs
	pool     []interface{}
	idle     *list.List
	mu       sync.RWMutex
	initFunc StartPool
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
	pools.idle = list.New()
	pools.pool = make([]interface{}, 0)
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

	if pool == nil {
		return nil, errors.New("cant find pool")
	}

	method, found := reflect.TypeOf(pool).MethodByName(funcName)
	if !found {
		return nil, errors.New("cant find method")
	}

	res := method.Func.Call([]reflect.Value{reflect.ValueOf(pool), reflect.ValueOf(args).Elem()})

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

func (pools *pools) CheckOut() interface{} {
	ch := make(chan interface{})
	timeOutChan := make(chan bool)

	go pools.checkout(ch, timeOutChan)

	select {
	case pool := <-ch:
		return pool
	case <-timeOutChan:
		return nil
	}
}

func (pools *pools) checkout(ch chan interface{}, timeOutChan chan bool) {
	pools.mu.Lock()
	if pools.idle.Len() > 0 {
		poolElement := pools.idle.Front()
		pools.idle.Remove(poolElement)
		pool := poolElement.Value

		ch <- pool
		pools.mu.Unlock()
	} else {
		if len(pools.pool) < pools.Size {
			pools.newPool()
			poolElement := pools.idle.Front()
			pools.idle.Remove(poolElement)
			pool := poolElement.Value

			ch <- pool
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
					if pools.idle.Len() > 0 {
						poolElement := pools.idle.Front()
						pools.idle.Remove(poolElement)
						pool := poolElement.Value

						ch <- pool
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

func (pools *pools) CheckIn(pool interface{}) {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	pools.idle.PushBack(pool)
}

func (pools *pools) newPool() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	pool, err := pools.initFunc()
	if err != nil {
		return err
	}

	pools.pool = append(pools.pool, pool)
	pools.idle.PushBack(pool)
	return nil
}