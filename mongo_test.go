package main

import (
	"fmt"
	"server/options"
	"testing"

	"github.com/PerforMance308/go-pool/go-pool"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type query struct {
	db     string
	table  string
	q      interface{}
	update interface{}
	res    interface{}
}

type session struct {
	dbConf options.DBConf
	s      *mgo.Session
}

type testData struct {
	A int
	B string
}

func TestStart(t *testing.T) {
	startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}
}

func TestInsert(t *testing.T) {
	startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}

	q := &query{
		table: "test",
		q:     testData{1, "2"},
	}
	if _, err := go_pool.Transaction("Insert", q); err != nil {
		t.Error("insert error")
	}

	t.Log("insert ok!!")
}

func TestFind(t *testing.T) {
	startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}

	q := &query{
		table: "test",
		q:     bson.M{"a": 1},
		res:   &testData{},
	}
	if _, err := go_pool.Transaction("Find", q); err != nil {
		t.Error("find error")
	}

	t.Log("find ok!!")
}

func TestUpdate(t *testing.T) {
	startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}

	q := &query{
		table:  "test",
		q:      bson.M{"a": 1},
		update: bson.M{"$set": bson.M{"b": "3"}},
	}
	if _, err := go_pool.Transaction("Update", q); err != nil {
		t.Error("update error")
	}

	t.Log("update ok!!")
}

func TestRemove(t *testing.T) {
	startFunc := func() (interface{}, error) {
		return Start()
	}

	if err := go_pool.Start(go_pool.PoolArgs{Size: 10}, startFunc); err != nil {
		t.Error("go pool start error")
	}

	q := &query{
		table: "test",
		q:     bson.M{"a": 1},
	}
	if _, err := go_pool.Transaction("Remove", q); err != nil {
		t.Error("remove error")
	}

	t.Log("remove ok!!")
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

func (s *session) Insert(q query) error {
	return s.s.DB(s.dbConf.Name).C(q.table).Insert(q.q)
}

func (s *session) Find(q query) error {
	return s.s.DB(s.dbConf.Name).C(q.table).Find(q.q).One(q.res)
}

func (s *session) Update(q query) error {
	return s.s.DB(s.dbConf.Name).C(q.table).Update(q.q, q.update)
}

func (s *session) Remove(q query) error {
	return s.s.DB(s.dbConf.Name).C(q.table).Remove(q.q)
}
