package GoMybatis

import "github.com/timandy/routine"

var sessionMapTls = routine.NewThreadLocal[Session]()

type GoroutineSessionMap struct {
}

func NewGoroutineSessionMap() *GoroutineSessionMap {
	return &GoroutineSessionMap{}
}

func (it *GoroutineSessionMap) Put(session Session) {
	sessionMapTls.Set(session)
}

func (it *GoroutineSessionMap) Get() Session {
	return sessionMapTls.Get()
}

func (it *GoroutineSessionMap) Delete() {
	sessionMapTls.Remove()
}
