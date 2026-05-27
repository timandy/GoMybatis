package stmt

import "fmt"

var _ StmtIndexConvertReusable = (*PostgreStmtIndexConvertImpl)(nil)

type PostgreStmtIndexConvertImpl struct {
	counter int
	cache   map[string]string
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

func (p *PostgreStmtIndexConvertImpl) Lookup(name string) (string, bool) {
	s, ok := p.cache[name]
	return s, ok
}

func (p *PostgreStmtIndexConvertImpl) Register(name string, placeholder string) {
	if p.cache == nil {
		p.cache = make(map[string]string, 4)
	}
	p.cache[name] = placeholder
}
