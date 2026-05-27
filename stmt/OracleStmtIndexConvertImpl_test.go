package stmt

import "testing"

func TestOracleStmtIndexConvertImpl_Convert(t *testing.T) {
	var convert = &OracleStmtIndexConvertImpl{}
	convert.Inc()
	if " :val1 " != convert.Convert() {
		panic("TestOracleStmtIndexConvertImpl_Convert fail")
	}
}

func TestOracleStmtIndexConvertImpl_LookupMiss(t *testing.T) {
	var c = &OracleStmtIndexConvertImpl{}
	if b, ok := c.Lookup("any"); ok || b != nil {
		t.Fatalf("expected miss on empty cache, got %q ok=%v", b, ok)
	}
}

func TestOracleStmtIndexConvertImpl_RegisterThenLookup(t *testing.T) {
	var c = &OracleStmtIndexConvertImpl{}
	c.Register("id", []byte(" :val1 "))
	if b, ok := c.Lookup("id"); !ok || string(b) != " :val1 " {
		t.Fatalf("expected hit returning \" :val1 \", got %q ok=%v", b, ok)
	}
}

func TestOracleStmtIndexConvertImpl_SatisfiesReusable(t *testing.T) {
	var c StmtIndexConvert = &OracleStmtIndexConvertImpl{}
	if _, ok := c.(StmtIndexConvertReusable); !ok {
		t.Fatal("OracleStmtIndexConvertImpl must satisfy StmtIndexConvertReusable")
	}
}
