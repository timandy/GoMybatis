package stmt

import "fmt"

var _ StmtIndexConvertReusable = (*OracleStmtIndexConvertImpl)(nil)

type OracleStmtIndexConvertImpl struct {
	counter int
	cache   map[string]string
}

func (it *OracleStmtIndexConvertImpl) Convert() string {
	return fmt.Sprint(" :val", it.Get(), " ")
}

func (it *OracleStmtIndexConvertImpl) Inc() {
	it.counter++
}

func (it *OracleStmtIndexConvertImpl) Get() int {
	return it.counter
}

func (it *OracleStmtIndexConvertImpl) Lookup(name string) (string, bool) {
	s, ok := it.cache[name]
	return s, ok
}

func (it *OracleStmtIndexConvertImpl) Register(name string, placeholder string) {
	if it.cache == nil {
		it.cache = make(map[string]string, 4)
	}
	it.cache[name] = placeholder
}
