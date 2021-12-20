package GoMybatis

import (
	"github.com/timandy/GoMybatis/v7/ast"
	"github.com/timandy/GoMybatis/v7/stmt"
)

//sql文本构建
type SqlBuilder interface {
	BuildSql(paramMap map[string]interface{}, nodes []ast.Node, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) (string, error)
	ExpressionEngineProxy() *ExpressionEngineProxy
	SetEnableLog(enable bool)
	EnableLog() bool
	NodeParser() ast.NodeParser
}
