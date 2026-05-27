package ast

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timandy/GoMybatis/v7/stmt"
)

//stubEngine 是 ExpressionEngine 的最小测试桩。
//LexerAndEval 行为可通过 lexerFn 配置：返回值用于模拟"env 中找不到 name 时的回退求值"。
type stubEngine struct {
	lexerFn func(expr string, arg interface{}) (interface{}, error)
}

func (s *stubEngine) Name() string { return "stub" }
func (s *stubEngine) Lexer(lexerArg string) (interface{}, error) {
	return nil, nil
}
func (s *stubEngine) Eval(lexerResult interface{}, arg interface{}, operation int) (interface{}, error) {
	return nil, nil
}
func (s *stubEngine) LexerAndEval(expr string, arg interface{}) (interface{}, error) {
	if s.lexerFn != nil {
		return s.lexerFn(expr, arg)
	}
	return nil, errors.New("not configured")
}

func newHolder(fn func(string, interface{}) (interface{}, error)) *NodeConfigHolder {
	return &NodeConfigHolder{Proxy: &stubEngine{lexerFn: fn}}
}

//--- tokenize ----------------------------------------------------------------

func TestTokenize_Empty(t *testing.T) {
	assert.Empty(t, tokenize("", nil))
}

func TestTokenize_PureLiteral(t *testing.T) {
	tokens := tokenize("select * from t", nil)
	assert.Len(t, tokens, 1)
	raw, ok := tokens[0].(*RawToken)
	assert.True(t, ok)
	assert.Equal(t, "select * from t", raw.text)
}

func TestTokenize_SingleExpr(t *testing.T) {
	holder := newHolder(nil)
	tokens := tokenize("id=#{id}", holder)
	assert.Len(t, tokens, 2)

	r, ok := tokens[0].(*RawToken)
	assert.True(t, ok)
	assert.Equal(t, "id=", r.text)

	e, ok := tokens[1].(*ExprToken)
	assert.True(t, ok)
	assert.Equal(t, "id", e.name)
	assert.Same(t, holder, e.holder)
}

func TestTokenize_SingleRawExpr(t *testing.T) {
	holder := newHolder(nil)
	tokens := tokenize("order by ${col}", holder)
	assert.Len(t, tokens, 2)

	e, ok := tokens[1].(*RawExprToken)
	assert.True(t, ok)
	assert.Equal(t, "col", e.name)
	assert.Same(t, holder, e.holder)
}

func TestTokenize_MultipleAndMixed(t *testing.T) {
	holder := newHolder(nil)
	tokens := tokenize("a #{x} b ${y} c #{z}", holder)
	// 期望: Raw("a "), Expr("x"), Raw(" b "), RawExpr("y"), Raw(" c "), Expr("z")
	assert.Len(t, tokens, 6)
	assert.IsType(t, &RawToken{}, tokens[0])
	assert.IsType(t, &ExprToken{}, tokens[1])
	assert.IsType(t, &RawToken{}, tokens[2])
	assert.IsType(t, &RawExprToken{}, tokens[3])
	assert.IsType(t, &RawToken{}, tokens[4])
	assert.IsType(t, &ExprToken{}, tokens[5])

	assert.Equal(t, "a ", tokens[0].(*RawToken).text)
	assert.Equal(t, "x", tokens[1].(*ExprToken).name)
	assert.Equal(t, " b ", tokens[2].(*RawToken).text)
	assert.Equal(t, "y", tokens[3].(*RawExprToken).name)
	assert.Equal(t, " c ", tokens[4].(*RawToken).text)
	assert.Equal(t, "z", tokens[5].(*ExprToken).name)
}

func TestTokenize_CommaStripped(t *testing.T) {
	holder := newHolder(nil)
	tokens := tokenize("#{name, jdbcType=VARCHAR}", holder)
	assert.Len(t, tokens, 1)
	assert.Equal(t, "name", tokens[0].(*ExprToken).name)
}

func TestTokenize_UnclosedPlaceholder(t *testing.T) {
	tokens := tokenize("select #{unfinished", nil)
	assert.Len(t, tokens, 1)
	assert.Equal(t, "select #{unfinished", tokens[0].(*RawToken).text)
}

func TestTokenize_HashWithoutBrace(t *testing.T) {
	tokens := tokenize("count(#) from t", nil)
	assert.Len(t, tokens, 1)
	assert.Equal(t, "count(#) from t", tokens[0].(*RawToken).text)
}

//--- RawToken.Render ---------------------------------------------------------

func TestRawToken_Render(t *testing.T) {
	r := &RawToken{text: "hello"}
	b, err := r.Render(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(b))
}

//--- ExprToken.Render --------------------------------------------------------

func TestExprToken_Render_ScalarFromEnv(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "id", holder: holder}
	env := map[string]interface{}{"id": 42}
	arr := []interface{}{}
	b, err := tok.Render(env, &arr, &stmt.PostgreStmtIndexConvertImpl{})
	assert.NoError(t, err)
	// PG Convert 返回 " $1 "（含前后空格），ExprToken 不应额外加工
	assert.Equal(t, " $1 ", string(b))
	assert.Equal(t, []interface{}{42}, arr)
}

func TestExprToken_Render_LexerFallback(t *testing.T) {
	// env 中没有 name 时, 应回退到 engine.LexerAndEval
	holder := newHolder(func(expr string, arg interface{}) (interface{}, error) {
		assert.Equal(t, "x", expr)
		return "lexed-value", nil
	})
	tok := &ExprToken{name: "x", holder: holder}
	arr := []interface{}{}
	b, err := tok.Render(map[string]interface{}{}, &arr, &stmt.MysqlStmtIndexConvertImpl{})
	assert.NoError(t, err)
	assert.Equal(t, " ? ", string(b))
	assert.Equal(t, []interface{}{"lexed-value"}, arr)
}

func TestExprToken_Render_LexerError(t *testing.T) {
	holder := newHolder(func(expr string, arg interface{}) (interface{}, error) {
		return nil, errors.New("boom")
	})
	tok := &ExprToken{name: "x", holder: holder}
	arr := []interface{}{}
	_, err := tok.Render(map[string]interface{}{}, &arr, &stmt.MysqlStmtIndexConvertImpl{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	assert.Empty(t, arr)
}

func TestExprToken_Render_Slice(t *testing.T) {
	holder := newHolder(nil)
	tok := &ExprToken{name: "ids", holder: holder}
	env := map[string]interface{}{"ids": []int{10, 20, 30}}
	arr := []interface{}{}
	b, err := tok.Render(env, &arr, &stmt.PostgreStmtIndexConvertImpl{})
	assert.NoError(t, err)
	// PG Convert 返回 " $N ", 用 ',' 连接, 整体 ( ) 包裹 —— 与旧 NodeForEach 借用展开一致
	assert.Equal(t, "( $1 , $2 , $3 )", string(b))
	assert.Equal(t, []interface{}{10, 20, 30}, arr)
}

//--- RawExprToken.Render -----------------------------------------------------

func TestRawExprToken_Render_FromEnv(t *testing.T) {
	holder := newHolder(nil)
	tok := &RawExprToken{name: "col", holder: holder}
	env := map[string]interface{}{"col": "user_name"}
	b, err := tok.Render(env, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "user_name", string(b))
}

func TestRawExprToken_Render_LexerFallback(t *testing.T) {
	holder := newHolder(func(expr string, arg interface{}) (interface{}, error) {
		return 123, nil
	})
	tok := &RawExprToken{name: "z", holder: holder}
	b, err := tok.Render(map[string]interface{}{}, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "123", string(b))
}

func TestRawExprToken_Render_LexerError(t *testing.T) {
	holder := newHolder(func(expr string, arg interface{}) (interface{}, error) {
		return nil, errors.New("kaboom")
	})
	tok := &RawExprToken{name: "z", holder: holder}
	_, err := tok.Render(map[string]interface{}{}, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kaboom")
}

//--- NodeString --------------------------------------------------------------

func TestNodeString_Type(t *testing.T) {
	n := &NodeString{}
	assert.Equal(t, NString, n.Type())
}

func TestNodeString_Eval_Empty(t *testing.T) {
	n := &NodeString{}
	b, err := n.Eval(nil, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, b)
}

func TestNodeString_Eval_PropagatesError(t *testing.T) {
	holder := newHolder(func(expr string, arg interface{}) (interface{}, error) {
		return nil, errors.New("eval-err")
	})
	n := &NodeString{
		tokens: []Token{
			&RawToken{text: "x="},
			&ExprToken{name: "missing", holder: holder},
		},
		holder: holder,
	}
	arr := []interface{}{}
	_, err := n.Eval(map[string]interface{}{}, &arr, &stmt.MysqlStmtIndexConvertImpl{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "eval-err")
}

func TestNodeString_Eval_Composite(t *testing.T) {
	holder := newHolder(nil)
	n := &NodeString{
		tokens: []Token{
			&RawToken{text: "x="},
			&ExprToken{name: "v", holder: holder},
		},
		holder: holder,
	}
	arr := []interface{}{}
	b, err := n.Eval(map[string]interface{}{"v": 7}, &arr, &stmt.MysqlStmtIndexConvertImpl{})
	assert.NoError(t, err)
	// "x=" + " ? " (MySQL Convert 含前后空格)
	assert.Equal(t, "x= ? ", string(b))
	assert.Equal(t, []interface{}{7}, arr)
}

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
