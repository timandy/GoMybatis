package engines

import (
	"testing"

	"github.com/timandy/GoMybatis/v7/example"
)

func BenchmarkExpressionEngineGoExpress_Eval(b *testing.B) {
	b.StopTimer()
	var activity = example.Activity{
		Id:         "1",
		DeleteFlag: 1,
	}

	var engine = ExpressionEngineGoExpress{}
	var evaluateParameters = make(map[string]interface{})

	evaluateParameters["activity"] = &activity
	var expression = "activity.DeleteFlag == 1 and activity.DeleteFlag != 0 "
	evalExpression, err := engine.Lexer(expression)
	if err != nil {
		b.Fatal(err)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Eval(evalExpression, evaluateParameters, 0)
		if err != nil {
			b.Fatal(err)
		}
		//fmt.Println(r)
	}
}
