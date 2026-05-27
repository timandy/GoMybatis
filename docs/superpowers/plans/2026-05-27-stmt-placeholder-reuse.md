# Postgres/Oracle Placeholder Reuse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Postgres / Oracle reuse the same `$N` / `:valN` placeholder when the same `#{name}` appears multiple times in one query, while leaving MySQL untouched. Also remove dead `RegexReplaceArg.go` code and the unused locks on the Postgres / Oracle stmt converters.

**Architecture:** Add a new optional interface `StmtIndexConvertReusable { Lookup; Register }` in the `stmt` package. Postgres and Oracle converters implement it and hold a `cache map[string]string`. `ExprToken.Render` does a type assertion: on cache hit return the cached placeholder verbatim (no append, no Inc, no LexerAndEval); on miss it renders as today and then `Register`s the result. MySQL keeps the old behavior via type-assertion failure → automatic fallback.

**Tech Stack:** Go 1.x, testify `assert`. No external deps added.

**Reference Spec:** `docs/superpowers/specs/2026-05-27-stmt-placeholder-reuse-design.md`

---

## File Map

- **Delete:** `ast/RegexReplaceArg.go`, `ast/RegexReplaceArg_test.go`
- **Modify:** `stmt/StmtIndexConvert.go` — add `StmtIndexConvertReusable` interface
- **Modify:** `stmt/PostgreStmtIndexConvertImpl.go` — drop mutex, add cache + Lookup/Register
- **Modify:** `stmt/OracleStmtIndexConvertImpl.go` — drop mutex, add cache + Lookup/Register
- **Modify:** `stmt/PostgreStmtIndexConvertImpl_test.go` — add coverage for Lookup/Register
- **Modify:** `stmt/OracleStmtIndexConvertImpl_test.go` — add coverage for Lookup/Register
- **Modify:** `ast/Token.go` — `ExprToken.Render` consults the reuse cache
- **Modify:** `ast/Token_test.go` — full test matrix (scalar/slice reuse on PG+Oracle, MySQL regression, empty slice)

---

## Task 1: Remove dead RegexReplaceArg.go

**Files:**
- Delete: `ast/RegexReplaceArg.go`
- Delete: `ast/RegexReplaceArg_test.go`

- [ ] **Step 1: Verify no external references remain**

Run:
```
grep -rn "FindExpress\|FindRawExpressString" --include="*.go" .
```
Expected: only matches inside `ast/RegexReplaceArg.go` and `ast/RegexReplaceArg_test.go`. If any other match shows up, stop and re-evaluate — the file is still in use.

- [ ] **Step 2: Delete both files**

```
rm ast/RegexReplaceArg.go
rm ast/RegexReplaceArg_test.go
```

- [ ] **Step 3: Build and test to confirm no breakage**

Run:
```
go build ./...
go test ./...
```
Expected: build succeeds, all existing tests pass.

- [ ] **Step 4: Commit**

```
git add -A
git commit -m "chore: remove dead RegexReplaceArg.go and its tests

FindExpress / FindRawExpressString have had no production callers since
the NodeString token-stream refactor in 7ad0298. Replace / ReplaceRaw
were already removed in the same commit."
```

---

## Task 2: Add StmtIndexConvertReusable interface to the stmt package

**Files:**
- Modify: `stmt/StmtIndexConvert.go`

- [ ] **Step 1: Edit `stmt/StmtIndexConvert.go` to add the optional interface**

Replace the entire file with:

```go
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
```

- [ ] **Step 2: Build to confirm the file still compiles**

Run:
```
go build ./stmt/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```
git add stmt/StmtIndexConvert.go
git commit -m "feat(stmt): add StmtIndexConvertReusable optional interface"
```

---

## Task 3: Implement StmtIndexConvertReusable on Postgres converter (TDD, also drops the mutex)

**Files:**
- Modify: `stmt/PostgreStmtIndexConvertImpl.go`
- Modify: `stmt/PostgreStmtIndexConvertImpl_test.go`

- [ ] **Step 1: Add the failing test**

Append to `stmt/PostgreStmtIndexConvertImpl_test.go`:

```go
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
```

- [ ] **Step 2: Run new tests, confirm they fail**

Run:
```
go test ./stmt/ -run TestPostgreStmtIndexConvertImpl_LookupMiss -v
```
Expected: compile error — `Lookup` undefined on `PostgreStmtIndexConvertImpl`.

- [ ] **Step 3: Rewrite `stmt/PostgreStmtIndexConvertImpl.go`**

Replace the whole file with:

```go
package stmt

import "fmt"

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
```

- [ ] **Step 4: Run all stmt tests, confirm pass**

Run:
```
go test ./stmt/ -v
```
Expected: all tests pass including the new three and the pre-existing `TestPostgreStmtIndexConvertImpl_Convert`.

- [ ] **Step 5: Commit**

```
git add stmt/PostgreStmtIndexConvertImpl.go stmt/PostgreStmtIndexConvertImpl_test.go
git commit -m "feat(stmt): postgres converter implements StmtIndexConvertReusable

Drops the sync.RWMutex: a StmtIndexConvert instance is owned by one
query rendering pipeline (single goroutine), so the lock is dead
weight. Adds a lazy-init name->placeholder cache used by
ExprToken.Render to reuse \$N for duplicate #{} occurrences."
```

---

## Task 4: Implement StmtIndexConvertReusable on Oracle converter (mirrors Task 3)

**Files:**
- Modify: `stmt/OracleStmtIndexConvertImpl.go`
- Modify: `stmt/OracleStmtIndexConvertImpl_test.go`

- [ ] **Step 1: Add the failing test**

Append to `stmt/OracleStmtIndexConvertImpl_test.go`:

```go
func TestOracleStmtIndexConvertImpl_LookupMiss(t *testing.T) {
	var c = &OracleStmtIndexConvertImpl{}
	if s, ok := c.Lookup("any"); ok || s != "" {
		t.Fatalf("expected miss on empty cache, got %q ok=%v", s, ok)
	}
}

func TestOracleStmtIndexConvertImpl_RegisterThenLookup(t *testing.T) {
	var c = &OracleStmtIndexConvertImpl{}
	c.Register("id", " :val1 ")
	if s, ok := c.Lookup("id"); !ok || s != " :val1 " {
		t.Fatalf("expected hit returning \" :val1 \", got %q ok=%v", s, ok)
	}
}

func TestOracleStmtIndexConvertImpl_SatisfiesReusable(t *testing.T) {
	var c StmtIndexConvert = &OracleStmtIndexConvertImpl{}
	if _, ok := c.(StmtIndexConvertReusable); !ok {
		t.Fatal("OracleStmtIndexConvertImpl must satisfy StmtIndexConvertReusable")
	}
}
```

- [ ] **Step 2: Run new tests, confirm they fail**

Run:
```
go test ./stmt/ -run TestOracleStmtIndexConvertImpl_LookupMiss -v
```
Expected: compile error — `Lookup` undefined on `OracleStmtIndexConvertImpl`.

- [ ] **Step 3: Rewrite `stmt/OracleStmtIndexConvertImpl.go`**

Replace the whole file with:

```go
package stmt

import "fmt"

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
```

- [ ] **Step 4: Run all stmt tests, confirm pass**

Run:
```
go test ./stmt/ -v
```
Expected: all stmt tests pass.

- [ ] **Step 5: Commit**

```
git add stmt/OracleStmtIndexConvertImpl.go stmt/OracleStmtIndexConvertImpl_test.go
git commit -m "feat(stmt): oracle converter implements StmtIndexConvertReusable

Mirrors the postgres change: drops the unused mutex, adds lazy-init
name->placeholder cache."
```

---

## Task 5: Wire ExprToken.Render to consult the reuse cache (TDD)

**Files:**
- Modify: `ast/Token.go`
- Modify: `ast/Token_test.go`

- [ ] **Step 1: Add a failing test for scalar placeholder reuse on Postgres**

Append to `ast/Token_test.go`:

```go
func TestExprToken_ScalarReuse_Postgres(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "id", holder: holder}
	env := map[string]interface{}{"id": 42}
	var args []interface{}
	conv := &stmt.PostgreStmtIndexConvertImpl{}

	// 1st render: misses cache → appends, increments, returns " $1 "
	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " $1 ", string(b1))
	assert.Equal(t, []interface{}{42}, args)

	// 2nd render of the SAME name: hits cache → no append, no Inc
	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " $1 ", string(b2))
	assert.Equal(t, []interface{}{42}, args, "duplicate #{id} must not push another arg")
	assert.Equal(t, 1, conv.Get(), "counter must not advance on cache hit")
}
```

- [ ] **Step 2: Run the new test, confirm it fails**

Run:
```
go test ./ast/ -run TestExprToken_ScalarReuse_Postgres -v
```
Expected: FAIL — second call appends another arg and advances counter (current behavior pre-change).

- [ ] **Step 3: Rewrite `ast/Token.go` ExprToken.Render with cache check + Register**

Replace the `ExprToken.Render` function body (lines 39–68 of the current `ast/Token.go`) with:

```go
func (t *ExprToken) Render(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	reusable, _ := stmtConvert.(stmt.StmtIndexConvertReusable)
	if reusable != nil {
		if cached, ok := reusable.Lookup(t.name); ok {
			return []byte(cached), nil
		}
	}

	engine := t.holder.GetExpressionEngineProxy()
	argValue := env[t.name]
	if argValue == nil {
		var err error
		argValue, err = engine.LexerAndEval(t.name, env)
		if err != nil {
			return nil, errors.New(engine.Name() + ":" + err.Error())
		}
	}

	var rendered []byte
	v := reflect.ValueOf(argValue)
	if v.Kind() == reflect.Slice {
		//与旧版 NodeForEach 借用展开 ( elem1 , elem2 , elem3 ) 的形态保持等价
		var buf bytes.Buffer
		buf.WriteByte('(')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			*arg_array = append(*arg_array, v.Index(i).Interface())
			stmtConvert.Inc()
			buf.WriteString(stmtConvert.Convert())
		}
		buf.WriteByte(')')
		rendered = buf.Bytes()
	} else {
		*arg_array = append(*arg_array, argValue)
		stmtConvert.Inc()
		rendered = []byte(stmtConvert.Convert())
	}

	if reusable != nil {
		reusable.Register(t.name, string(rendered))
	}
	return rendered, nil
}
```

- [ ] **Step 4: Run the new test, confirm it passes**

Run:
```
go test ./ast/ -run TestExprToken_ScalarReuse_Postgres -v
```
Expected: PASS.

- [ ] **Step 5: Run the full ast suite to confirm no regression**

Run:
```
go test ./ast/ -v
```
Expected: all pre-existing tokenize / Token tests still pass.

- [ ] **Step 6: Commit**

```
git add ast/Token.go ast/Token_test.go
git commit -m "feat(ast): ExprToken.Render reuses cached placeholder by name

When the stmtConvert implements StmtIndexConvertReusable (postgres /
oracle), duplicate #{name} occurrences in one query share the same
placeholder string. arg_array is appended to and Inc() is called only
on the first render; subsequent renders return the cached bytes
verbatim. MySQL's converter does not implement the optional interface,
so behavior is unchanged for ? placeholders."
```

---

## Task 6: Add the full reuse / regression / empty-slice test matrix

**Files:**
- Modify: `ast/Token_test.go`

- [ ] **Step 1: Append all remaining test matrix cases**

Append to `ast/Token_test.go`:

```go
//--- 复用语义全矩阵 ---------------------------------------------------------

func TestExprToken_ScalarReuse_Oracle(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "id", holder: holder}
	env := map[string]interface{}{"id": 42}
	var args []interface{}
	conv := &stmt.OracleStmtIndexConvertImpl{}

	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " :val1 ", string(b1))

	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " :val1 ", string(b2))
	assert.Equal(t, []interface{}{42}, args)
	assert.Equal(t, 1, conv.Get())
}

func TestExprToken_ScalarNoReuse_MySQL(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "id", holder: holder}
	env := map[string]interface{}{"id": 42}
	var args []interface{}
	conv := &stmt.MysqlStmtIndexConvertImpl{}

	// MySQL ? cannot back-reference; both renders must independently append.
	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " ? ", string(b1))

	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, " ? ", string(b2))
	assert.Equal(t, []interface{}{42, 42}, args, "MySQL must keep duplicating args")
}

func TestExprToken_SliceReuse_Postgres(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{1, 2, 3}}
	var args []interface{}
	conv := &stmt.PostgreStmtIndexConvertImpl{}

	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( $1 , $2 , $3 )", string(b1))
	assert.Equal(t, []interface{}{1, 2, 3}, args)

	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( $1 , $2 , $3 )", string(b2), "second #{ids} must reuse the same group")
	assert.Equal(t, []interface{}{1, 2, 3}, args, "slice reuse must not re-append")
	assert.Equal(t, 3, conv.Get())
}

func TestExprToken_SliceReuse_Oracle(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{1, 2, 3}}
	var args []interface{}
	conv := &stmt.OracleStmtIndexConvertImpl{}

	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( :val1 , :val2 , :val3 )", string(b1))

	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( :val1 , :val2 , :val3 )", string(b2))
	assert.Equal(t, []interface{}{1, 2, 3}, args)
	assert.Equal(t, 3, conv.Get())
}

func TestExprToken_SliceNoReuse_MySQL(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{1, 2, 3}}
	var args []interface{}
	conv := &stmt.MysqlStmtIndexConvertImpl{}

	b1, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( ? , ? , ? )", string(b1))

	b2, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "( ? , ? , ? )", string(b2))
	assert.Equal(t, []interface{}{1, 2, 3, 1, 2, 3}, args,
		"MySQL must duplicate slice args for each occurrence")
}

func TestExprToken_MixedNames_Postgres(t *testing.T) {
	// Mirrors design doc §5: #{ids} reused, #{name} fresh; final counter advances normally.
	holder := newHolder(nil)
	idsTok := &ExprToken{name: "ids", holder: holder}
	nameTok := &ExprToken{name: "name", holder: holder}
	env := map[string]interface{}{
		"ids":  []int{1, 2, 3},
		"name": "foo",
	}
	var args []interface{}
	conv := &stmt.PostgreStmtIndexConvertImpl{}

	b1, _ := idsTok.Render(env, &args, conv)
	assert.Equal(t, "( $1 , $2 , $3 )", string(b1))
	b2, _ := idsTok.Render(env, &args, conv)
	assert.Equal(t, "( $1 , $2 , $3 )", string(b2))
	b3, _ := nameTok.Render(env, &args, conv)
	assert.Equal(t, " $4 ", string(b3))

	assert.Equal(t, []interface{}{1, 2, 3, "foo"}, args)
	assert.Equal(t, 4, conv.Get())
}

func TestExprToken_EmptySlice_Postgres(t *testing.T) {
	// Documented behavior: empty IN list renders as "()", produces invalid
	// SQL on most dialects, fail-loud at the driver. No args pushed, no Inc.
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{}}
	var args []interface{}
	conv := &stmt.PostgreStmtIndexConvertImpl{}

	b, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "()", string(b))
	assert.Empty(t, args)
	assert.Equal(t, 0, conv.Get())
}

func TestExprToken_EmptySlice_MySQL(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{}}
	var args []interface{}
	conv := &stmt.MysqlStmtIndexConvertImpl{}

	b, err := tok.Render(env, &args, conv)
	assert.NoError(t, err)
	assert.Equal(t, "()", string(b))
	assert.Empty(t, args)
}
```

- [ ] **Step 2: Run the full new matrix**

Run:
```
go test ./ast/ -run TestExprToken_ -v
```
Expected: all 9 new tests pass (plus the one from Task 5 — 10 total under that prefix that we authored).

- [ ] **Step 3: Run the entire repo test suite**

Run:
```
go build ./...
go test ./...
```
Expected: all packages build, all tests pass. If any pre-existing integration test relied on duplicate-#{} behavior on Postgres / Oracle, it will fail here — adjust the test (the new behavior is the correct one per spec §8).

- [ ] **Step 4: Commit**

```
git add ast/Token_test.go
git commit -m "test(ast): cover placeholder reuse matrix and empty-slice behavior

Locks in: scalar/slice reuse on Postgres+Oracle; MySQL regression
(no reuse, args duplicated); mixed name reuse + counter advancement;
empty slice renders to \"()\" with no args / no Inc."
```

---

## Self-Review Notes

**Spec coverage check** (against `docs/superpowers/specs/2026-05-27-stmt-placeholder-reuse-design.md`):

- §4.1 Interface — Task 2 ✓
- §4.2 Postgres / Oracle impl — Tasks 3, 4 ✓
- §4.3 ExprToken.Render — Task 5 ✓
- §4.4 Empty slice behavior — Task 6 (two empty-slice tests) ✓
- §5 Data flow example — Task 6 `TestExprToken_MixedNames_Postgres` ✓
- §6 Dead code cleanup — Task 1 ✓
- §7 Test matrix (all 9 rows) — Tasks 3, 4, 5, 6 ✓
- §8 Compatibility (MySQL unchanged) — Task 6 MySQL regression tests ✓

**Type / naming consistency**: `Lookup(name string) (string, bool)` and `Register(name, placeholder string)` are used identically in interface declaration, both impls, ExprToken consumer, and tests. `counter` and `cache` field names match across both impls.

**Placeholder scan**: no TODO / TBD / "implement later" — every code change shows the full target code.
