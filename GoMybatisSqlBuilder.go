package GoMybatis

import (
	"github.com/timandy/GoMybatis/v7/ast"
	"github.com/timandy/GoMybatis/v7/stmt"
	"strings"
)

type GoMybatisSqlBuilder struct {
	expressionEngineProxy ExpressionEngineProxy
	enableLog             bool
	nodeParser            ast.NodeParser
}

func (it *GoMybatisSqlBuilder) ExpressionEngineProxy() *ExpressionEngineProxy {
	return &it.expressionEngineProxy
}

func (it GoMybatisSqlBuilder) New(expressionEngine ExpressionEngineProxy, log Log, enableLog bool) GoMybatisSqlBuilder {
	it.expressionEngineProxy = expressionEngine
	it.enableLog = enableLog
	it.nodeParser = ast.NodeParser{
		Holder: ast.NodeConfigHolder{
			Proxy: &expressionEngine,
		},
	}
	return it
}

func (it *GoMybatisSqlBuilder) BuildSql(paramMap map[string]interface{}, nodes []ast.Node, arg_array *[]interface{}, stmtConvert stmt.StmtIndexConvert) (string, error) {
	//抽象语法树节点构建
	var sql, err = ast.DoChildNodes(nodes, paramMap, arg_array, stmtConvert)
	if err != nil {
		return "", err
	}

	return FormatSql(string(sql)), nil
}

func FormatSql(sql string) string {
	split := strings.Split(sql, " ")
	curIndex := 0
	for i := 0; i < len(split); i++ {
		curStr := split[i]
		if len(curStr) == 0 {
			continue
		}
		split[curIndex] = curStr
		curIndex++
	}
	return strings.Join(split[0:curIndex], " ")
}

func (it *GoMybatisSqlBuilder) SetEnableLog(enable bool) {
	it.enableLog = enable
}
func (it *GoMybatisSqlBuilder) EnableLog() bool {
	return it.enableLog
}

func (it *GoMybatisSqlBuilder) NodeParser() ast.NodeParser {
	return it.nodeParser
}
