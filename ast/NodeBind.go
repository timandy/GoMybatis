package ast

import "github.com/timandy/GoMybatis/v7/stmt"

type NodeBind struct {
	t NodeType

	name  string
	value string

	holder *NodeConfigHolder
}

func (it *NodeBind) Type() NodeType {
	return NBind
}

func (it *NodeBind) Eval(env map[string]interface{}, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) ([]byte, error) {
	if it.name == "" {
		panic(`[GoMybatis] element <bind name = ""> name can not be nil!`)
	}
	if it.value == "" {
		env[it.name] = it.value
		return nil, nil
	}
	if it.holder == nil {
		return nil, nil
	}
	result, err := it.holder.GetExpressionEngineProxy().LexerAndEval(it.value, env)
	if err != nil {
		//TODO send log bind fail
		return nil, err
	}
	env[it.name] = result

	return nil, nil
}
