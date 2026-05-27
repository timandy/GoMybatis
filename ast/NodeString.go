package ast

import (
	"bytes"

	"github.com/timandy/GoMybatis/v7/stmt"
)

//字符串节点：解析期被切分为 token 流，运行期按序渲染。
type NodeString struct {
	tokens []Token
	t      NodeType
	holder *NodeConfigHolder
}

func (it *NodeString) Type() NodeType {
	return NString
}

func (it *NodeString) Eval(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	if len(it.tokens) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	for _, tok := range it.tokens {
		b, err := tok.Render(env, arg_array, stmtConvert)
		if err != nil {
			return nil, err
		}
		if b != nil {
			buf.Write(b)
		}
	}
	return buf.Bytes(), nil
}
