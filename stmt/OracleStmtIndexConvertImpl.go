package stmt

import "fmt"

var _ StmtIndexConvertReusable = (*OracleStmtIndexConvertImpl)(nil)

type OracleStmtIndexConvertImpl struct {
	counter int
	cache   map[string][]byte
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

func (it *OracleStmtIndexConvertImpl) Lookup(name string) ([]byte, bool) {
	b, ok := it.cache[name]
	return b, ok
}

func (it *OracleStmtIndexConvertImpl) Register(name string, placeholder []byte) {
	if it.cache == nil {
		it.cache = make(map[string][]byte, 4)
	}
	it.cache[name] = placeholder
}
