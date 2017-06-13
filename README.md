# go-pool
Applicable to all database connection pools
# example for mongodb
```
package main

func main() {
  startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}
}

func Start() (interface{}, error) {
	connUrl := "127.0.0.1:27017/test"

	fmt.Println("Connecting Mongodb", connUrl)
	sess, err := mgo.Dial(connUrl)
	if err != nil {
		return nil, err
	}

	s := &session{
		s: sess,
	}
	return s, nil
}
```
  go_pool:Start/2 is used to start the connection pool,2 parameters are connection pool parameters and callback methods

```
type query struct {
	db       string
	table    string
	selector interface{}
	update   interface{}
	res      interface{}
}

func Insert(table string, data interface{}) error {
  q := &query{
		table:    table,
		selector: data,
	}
	if _, err := go_pool.Transaction("Insert", q); err != nil {
		return err
	}

	return nil
}

func (s *session) Insert(q query) error {
	return s.s.DB(s.dbConf.Name).C(q.table).Insert(q.selector)
}
```
  Use go_pool.Transaction/2 to call your function and Each call to a function requires a callback method.Transaction requires 2          parameters, first one is callback method name, the second one is callback method arguments.
  
  You can use this pool on redis,mysql and any other database. And it can also be used as an object pool.
