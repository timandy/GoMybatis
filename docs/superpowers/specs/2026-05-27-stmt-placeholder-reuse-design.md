# Postgres/Oracle 同名占位符复用 + StmtIndexConvert 清理

- 日期：2026-05-27
- 状态：设计已对齐，待实现
- 关联提交：master `ca9edc1`（HEAD），重构起点 `7ad0298`（NodeString → token 流）

## 1. 背景与动机

`7ad0298` 把 `NodeString` 从"字符串全量替换"重构为 Token 流，修了非空 slice + 标量混用时 Postgres/Oracle 上 `$N` / `:valN` 与 `arg_array` 错位的 bug。但还有两个相关问题：

1. **同名占位符不复用**：同一 query 里出现两次 `#{id}` 会被绑定两次（`$1, $2` + arg_array 两份相同的值）。Postgres/Oracle 的占位符语法本身支持 `$1` / `:val1` 重复引用，浪费了能力。
2. **`RegexReplaceArg.go` 死代码**：`Replace` / `ReplaceRaw` 已被 `7ad0298` 删除，剩下的 `FindExpress` / `FindRawExpressString` 在生产路径已无任何调用（仅自身测试引用）。

附带清理：Postgres/Oracle 转换器里的 `sync.RWMutex` 在单 goroutine 渲染路径上是过度防御，顺手去掉。

## 2. 目标

- 在 Postgres/Oracle 上，同 query 中同名 `#{name}`（标量和 slice 都算）只 append 一次参数、只占用一个 / 一组 placeholder 索引，重复出现时复用首次渲染的占位符串。
- 删除 `ast/RegexReplaceArg.go` 及其测试。
- 简化 Postgres/Oracle 转换器，去掉冗余锁。
- 行为对 MySQL 完全保持现状（`?` 不支持回指，无法复用）。

## 3. 非目标

- 转义语法 `\#{`（暂无需求）。
- `sync.Pool` 化 `bytes.Buffer` / Token slice（YAGNI）。
- 缓存 `ExpressionEngineProxy`（YAGNI）。
- 重写 `NodeForEach`（独立议题，超范围）。
- 移除 `StmtIndexConvert.Get()`（接口对外导出，移除是 breaking change，保守保留）。

## 4. 设计

### 4.1 新可选接口

`stmt/StmtIndexConvert.go`：

```go
type StmtIndexConvert interface {
    Convert() string
    Inc()
    Get() int
}

// 可选能力：支持按 name 复用占位符的方言实现此接口。
// 调用方对 StmtIndexConvert 做类型断言；命中 Lookup 时跳过 append/Inc，
// 直接用返回的字符串拼接 SQL。缓存生命周期 = 单个 query。
type StmtIndexConvertReusable interface {
    Lookup(name string) (string, bool)
    Register(name string, placeholder string)
}
```

契约：
- `Register(name, placeholder)` 由 `ExprToken.Render` 在首次完整渲染后调用，存入"可直接拼回 SQL 的整段字符串"——标量是 `" $1 "`，slice 是 `"( $1 , $2 , $3 )"`。
- `Lookup(name)` 命中即返回完整字符串；调用方必须直接复用、不再触发 append / Inc / 引擎求值。
- 缓存按 `StmtIndexConvert` 实例存活，即一次 query 渲染周期。

### 4.2 实现

`stmt/PostgreStmtIndexConvertImpl.go`：

```go
type PostgreStmtIndexConvertImpl struct {
    counter int
    cache   map[string]string
}

func (p *PostgreStmtIndexConvertImpl) Inc()                       { p.counter++ }
func (p *PostgreStmtIndexConvertImpl) Get() int                   { return p.counter }
func (p *PostgreStmtIndexConvertImpl) Convert() string            { return fmt.Sprint(" $", p.Get(), " ") }
func (p *PostgreStmtIndexConvertImpl) Lookup(name string) (string, bool) {
    s, ok := p.cache[name]
    return s, ok
}
func (p *PostgreStmtIndexConvertImpl) Register(name, placeholder string) {
    if p.cache == nil {
        p.cache = make(map[string]string, 4)
    }
    p.cache[name] = placeholder
}
```

变更点：
- 去掉 `sync.RWMutex`。
- 新增 `cache` 字段 + `Lookup` / `Register`。
- `Inc()` / `Get()` / `Convert()` 行为与现状等价（仅去锁）。

`stmt/OracleStmtIndexConvertImpl.go` 同形，`Convert()` 返回 `fmt.Sprint(" :val", it.Get(), " ")`。

`stmt/MysqlStmtIndexConvertImpl.go` **完全不变**，不实现 `StmtIndexConvertReusable`，类型断言失败 → 自动 fallback 到老路径。

### 4.3 ExprToken.Render 改造

`ast/Token.go`：

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

不变量：
- 命中缓存 ⇒ 不 append、不 Inc、不调引擎求值。
- 未命中 ⇒ 走完整渲染后 Register。
- MySQL 类型断言失败 ⇒ `reusable=nil` ⇒ 永远未命中且不 Register，行为 100% 与今天一致。

`RawExprToken`（`${...}`）不改——纯字面量拼接，无参数绑定，无所谓"复用"。

### 4.4 空 slice 行为

`#{ids}` 当 ids=`[]` 时，`ExprToken.Render` slice 分支输出 `()`，**不 append、不 Inc**。

- SQL 形态：`WHERE id IN ()` ——大多数方言上是语法错，到驱动层报错。
- **这是预期行为**：fail-loud，提示调用方在 Go 层 guard `len(ids) > 0`，或在 XML 用 `<if>` 包裹。
- 替代方案"输出 `(NULL)` 静默返回 0 行"被显式拒绝：会掩盖业务侧的"空集合"问题。
- 在 `ast/Token.go` 代码注释里固定这一语义。

## 5. 数据流示例

输入：`WHERE a IN #{ids} OR b IN #{ids} AND name = #{name}`，env=`{ids:[1,2,3], name:"foo"}`，Postgres。

| # | Token | 命中? | counter | arg_array | cache 快照 | 输出片段 |
|---|---|---|---|---|---|---|
| 1 | `WHERE a IN ` | — | 0 | `[]` | `{}` | `WHERE a IN ` |
| 2 | `#{ids}` | 否 | 3 | `[1,2,3]` | `{ids:"( $1 , $2 , $3 )"}` | `( $1 , $2 , $3 )` |
| 3 | ` OR b IN ` | — | 3 | `[1,2,3]` | … | ` OR b IN ` |
| 4 | `#{ids}` | 是 | 3 | `[1,2,3]` | … | `( $1 , $2 , $3 )` |
| 5 | ` AND name = ` | — | 3 | `[1,2,3]` | … | ` AND name = ` |
| 6 | `#{name}` | 否 | 4 | `[1,2,3,"foo"]` | `{ids:…, name:" $4 "}` | ` $4 ` |

最终 SQL：`WHERE a IN ( $1 , $2 , $3 ) OR b IN ( $1 , $2 , $3 ) AND name =  $4`
args：`[1, 2, 3, "foo"]`

对照 MySQL（同输入，无 reuse）：
SQL：`WHERE a IN ( ? , ? , ? ) OR b IN ( ? , ? , ? ) AND name =  ? `
args：`[1, 2, 3, 1, 2, 3, "foo"]`

## 6. 死代码清理

删除：
- `ast/RegexReplaceArg.go`
- `ast/RegexReplaceArg_test.go`

清理前 `grep` 全仓库确认 `FindExpress` / `FindRawExpressString` 除文件自身和测试外再无引用（已查过）。

## 7. 测试矩阵

`ast/Token_test.go` 新增：

| 用例 | DB 风格 | 输入 | 期望 SQL（去空格简记） | 期望 args |
|---|---|---|---|---|
| 标量复用 | Postgres | `#{id},#{id},#{name}` | `$1,$1,$2` | `[1,"a"]` |
| 标量复用 | Oracle | `#{id},#{id},#{name}` | `:val1,:val1,:val2` | `[1,"a"]` |
| 标量回归 | MySQL | `#{id},#{id},#{name}` | `?,?,?` | `[1,1,"a"]` |
| slice 整段复用 | Postgres | `IN #{ids} OR IN #{ids}` | `IN ($1,$2,$3) OR IN ($1,$2,$3)` | `[1,2,3]` |
| slice 整段复用 | Oracle | 同上 | `IN (:val1,:val2,:val3) OR IN (:val1,:val2,:val3)` | `[1,2,3]` |
| slice 回归 | MySQL | 同上 | `IN (?,?,?) OR IN (?,?,?)` | `[1,2,3,1,2,3]` |
| 跨 NodeString 复用 | Postgres | XML: `SELECT #{id} <if test="..">#{id}</if>` | 两处 `#{id}` 同 `$1` | `[5]` |
| 空 slice | Postgres | `IN #{ids}` (ids=[]) | `IN ()` | `[]` |
| 空 slice | MySQL | 同上 | `IN ()` | `[]` |

清理项回归：`go build ./...` + 全量 `go test ./...` 通过即兜底。

## 8. 兼容性影响

- **MySQL 用户**：无任何变化。
- **Postgres / Oracle 用户**：同名占位符行为变更。如果业务代码依赖"重复 `#{id}` 必然 push 两份相同参数到 `arg_array`"，需要适配——但这种依赖本身罕见且偏离 SQL 语义，预期影响极小。
- **第三方自定义方言**：继续实现 `StmtIndexConvert` 即可工作；如果想要复用能力，额外实现 `StmtIndexConvertReusable`。
- **`FindExpress` / `FindRawExpressString` 外部调用方**：被删除。仓库内已搜索确认无引用；外部依赖不可知。这是一个明确的 breaking change，需要在 release notes / CHANGELOG 标注。

## 9. 风险与回退

- **风险**：`ExprToken.Render` 命中缓存路径与未命中路径需保持渲染等价；如果 Register 写入的字符串和 Convert() 直接输出的不一致，会让首次和后续 SQL 形态不同。缓解：单元测试覆盖"首次/复用渲染字节级一致"。
- **回退**：撤回 commit 即恢复 `7ad0298` 之后的行为。`StmtIndexConvertReusable` 是新增接口、可选实现，移除不会留下悬挂引用。

## 10. 实现顺序

1. 删除 `RegexReplaceArg.go` + 测试，单独 commit
2. 加 `StmtIndexConvertReusable` 接口 + Postgres/Oracle 实现（含去锁），单独 commit
3. 改 `ExprToken.Render`，单独 commit
4. 补齐 `ast/Token_test.go` 用例，单独 commit
5. 全量 `go test ./...`，确认绿
