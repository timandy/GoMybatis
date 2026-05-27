package stmt

import "testing"

func TestPostgreStmtIndexConvertImpl_Convert(t *testing.T) {
	var convert = &PostgreStmtIndexConvertImpl{}
	convert.Inc()
	convert.Inc()
	if " $2 " != convert.Convert() {
		panic("TestPostgreStmtIndexConvertImpl_Convert fail")
	}
}

func TestPostgreStmtIndexConvertImpl_LookupMiss(t *testing.T) {
	var c = &PostgreStmtIndexConvertImpl{}
	if b, ok := c.Lookup("any"); ok || b != nil {
		t.Fatalf("expected miss on empty cache, got %q ok=%v", b, ok)
	}
}

func TestPostgreStmtIndexConvertImpl_RegisterThenLookup(t *testing.T) {
	var c = &PostgreStmtIndexConvertImpl{}
	c.Register("id", []byte(" $1 "))
	if b, ok := c.Lookup("id"); !ok || string(b) != " $1 " {
		t.Fatalf("expected hit returning \" $1 \", got %q ok=%v", b, ok)
	}
}

func TestPostgreStmtIndexConvertImpl_SatisfiesReusable(t *testing.T) {
	var c StmtIndexConvert = &PostgreStmtIndexConvertImpl{}
	if _, ok := c.(StmtIndexConvertReusable); !ok {
		t.Fatal("PostgreStmtIndexConvertImpl must satisfy StmtIndexConvertReusable")
	}
}
