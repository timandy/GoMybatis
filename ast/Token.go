package ast

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/timandy/GoMybatis/v7/stmt"
)

//Token 是 NodeString 解析后的最小渲染单元。
//每个 Token 负责自己那一段 SQL 文本的产出，并按需追加参数到 arg_array、
//调用 stmtConvert.Inc 生成占位符。
//
//这套结构取代了旧版基于 strings.Replace 的"全量替换 + 多次 append"流程，
//从根本上消除了 PostgreSQL/Oracle 编号占位符与参数数组数量不一致的 bug。
type Token interface {
	Render(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error)
}

//RawToken 输出固定字面量片段，不消费任何参数。
type RawToken struct {
	text string
}

func (t *RawToken) Render(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	return []byte(t.text), nil
}

//ExprToken 对应 #{name}：把 env[name] 追加到 arg_array，并产出占位符。
//若 name 解析出的值是 slice，则就地展开为 (?, ?, ?) 形式（每个元素独立 append + Inc）。
type ExprToken struct {
	name   string
	holder *NodeConfigHolder
}

func (t *ExprToken) Render(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	engine := t.holder.GetExpressionEngineProxy()
	argValue := env[t.name]
	if argValue == nil {
		var err error
		argValue, err = engine.LexerAndEval(t.name, env)
		if err != nil {
			return nil, errors.New(engine.Name() + ":" + err.Error())
		}
	}
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
		return buf.Bytes(), nil
	}
	*arg_array = append(*arg_array, argValue)
	stmtConvert.Inc()
	return []byte(stmtConvert.Convert()), nil
}

//RawExprToken 对应 ${name}：把 env[name] 的字符串形式直接拼到 SQL，不走参数绑定。
type RawExprToken struct {
	name   string
	holder *NodeConfigHolder
}

func (t *RawExprToken) Render(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	engine := t.holder.GetExpressionEngineProxy()
	var evalData interface{}
	if v := env[t.name]; v != nil {
		evalData = v
	} else {
		var err error
		evalData, err = engine.LexerAndEval(t.name, env)
		if err != nil {
			return nil, errors.New(engine.Name() + ":" + err.Error())
		}
	}
	return []byte(fmt.Sprint(evalData)), nil
}

//tokenize 将原始模板字符串切分为 Token 序列。
//识别 #{...} 和 ${...} 两种占位符；其余文本作为 RawToken 原样保留。
//占位符内部出现 ',' 时取逗号前的部分作为 name（兼容 #{name, jdbcType=...} 写法），
//与历史 FindExpress / FindRawExpressString 的语义一致。
func tokenize(s string, holder *NodeConfigHolder) []Token {
	var tokens []Token
	n := len(s)
	i := 0
	rawStart := 0

	flush := func(end int) {
		if end > rawStart {
			tokens = append(tokens, &RawToken{text: s[rawStart:end]})
		}
	}

	for i < n {
		c := s[i]
		if (c == '#' || c == '$') && i+1 < n && s[i+1] == '{' {
			// 找到 }
			rel := strings.IndexByte(s[i+2:], '}')
			if rel < 0 {
				i++
				continue
			}
			end := i + 2 + rel
			content := s[i+2 : end]
			if idx := strings.IndexByte(content, ','); idx >= 0 {
				content = content[:idx]
			}
			flush(i)
			if c == '#' {
				tokens = append(tokens, &ExprToken{name: content, holder: holder})
			} else {
				tokens = append(tokens, &RawExprToken{name: content, holder: holder})
			}
			i = end + 1
			rawStart = i
			continue
		}
		i++
	}
	flush(n)
	return tokens
}
