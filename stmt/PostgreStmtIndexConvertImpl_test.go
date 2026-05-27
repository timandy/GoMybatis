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
	if s, ok := c.Lookup("any"); ok || s != "" {
		t.Fatalf("expected miss on empty cache, got %q ok=%v", s, ok)
	}
}

func TestPostgreStmtIndexConvertImpl_RegisterThenLookup(t *testing.T) {
	var c = &PostgreStmtIndexConvertImpl{}
	c.Register("id", " $1 ")
	if s, ok := c.Lookup("id"); !ok || s != " $1 " {
		t.Fatalf("expected hit returning \" $1 \", got %q ok=%v", s, ok)
	}
}

func TestPostgreStmtIndexConvertImpl_SatisfiesReusable(t *testing.T) {
	var c StmtIndexConvert = &PostgreStmtIndexConvertImpl{}
	if _, ok := c.(StmtIndexConvertReusable); !ok {
		t.Fatal("PostgreStmtIndexConvertImpl must satisfy StmtIndexConvertReusable")
	}
}
