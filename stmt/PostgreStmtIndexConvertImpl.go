package stmt

import "fmt"

var _ StmtIndexConvertReusable = (*PostgreStmtIndexConvertImpl)(nil)

type PostgreStmtIndexConvertImpl struct {
	counter int
	cache   map[string][]byte
}

func (p *PostgreStmtIndexConvertImpl) Inc() {
	p.counter++
}

func (p *PostgreStmtIndexConvertImpl) Get() int {
	return p.counter
}

func (p *PostgreStmtIndexConvertImpl) Convert() string {
	return fmt.Sprint(" $", p.Get(), " ")
}

func (p *PostgreStmtIndexConvertImpl) Lookup(name string) ([]byte, bool) {
	b, ok := p.cache[name]
	return b, ok
}

func (p *PostgreStmtIndexConvertImpl) Register(name string, placeholder []byte) {
	if p.cache == nil {
		p.cache = make(map[string][]byte, 4)
	}
	p.cache[name] = placeholder
}
