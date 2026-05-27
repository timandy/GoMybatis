package stmt

// stmt convert
// example mysql: input 1 -> ?
// oracle : input 0 ->   :val1
// sqlite: input 0 ->   " ? "
type StmtIndexConvert interface {
	Convert() string
	Inc()
	Get() int
}

// StmtIndexConvertReusable is an optional capability for placeholder
// dialects whose syntax supports back-reference (e.g. Postgres $N,
// Oracle :valN). When a StmtIndexConvert also satisfies this
// interface, callers may consult Lookup before rendering a new
// placeholder; on a cache hit they MUST skip arg append / Inc and
// reuse the returned string verbatim. The cache lives for one query
// (i.e. the lifetime of the StmtIndexConvert instance).
type StmtIndexConvertReusable interface {
	Lookup(name string) (string, bool)
	Register(name string, placeholder string)
}
