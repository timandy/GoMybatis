package GoMybatis

import (
	"github.com/zhuxiujia/GoMybatis/tx"
	"github.com/zhuxiujia/GoMybatis/utils"
	"reflect"
	"strings"
)

//使用AOP切面 代理目标服务，如果服务painc()它的事务会回滚
func AopProxyService(service reflect.Value, engine *GoMybatisEngine) {
	//调用方法栈
	var beanType = service.Type().Elem()
	var beanName = beanType.PkgPath() + beanType.Name()
	//var methodStackMap = tx.GoroutineMethodStackMap{}.New()
	ProxyValue(service, func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value {
		//init data
		var propagation = tx.PROPAGATION_NEVER
		var nativeImplFunc = reflect.ValueOf(field.Interface())
		var txTag, haveTx = funcField.Tag.Lookup("tx")
		var rollbackTag = funcField.Tag.Get("rollback")
		if haveTx {
			propagation = tx.NewPropagation(txTag)
		}
		var fn = func(arg ProxyArg) []reflect.Value {
			var goroutineID = utils.GoroutineID() //协程id
			//var methodStack = methodStackMap.Get(goroutineID) //方法栈
			//if methodStack == nil {
			//	var sf = tx.StructField{}.New()
			//	methodStack = &sf
			//}
			//methodStack.Push(funcField)

			var session = engine.GoroutineSessionMap().Get(goroutineID)
			if session == nil {
				//todo newSession is use service bean name?
				var err error
				session, err = engine.NewSession(beanName)
				if err != nil {
					panic(err)
				}
				//压入map
				engine.GoroutineSessionMap().Put(goroutineID, session)
			}
			//if methodStack.Len() == 1 {
			var err = session.Begin(&propagation)
			if err != nil {
				panic(err)
			}
			//}
			var nativeImplResult = doNativeMethod(arg, nativeImplFunc, session)
			//methodStack.Pop()
			//if methodStack.Len() == 0 && session != nil {
			if !haveRollBackType(nativeImplResult, rollbackTag) {
				var err = session.Commit()
				if err != nil {
					panic(err)
				}
			} else {
				var err = session.Rollback()
				if err != nil {
					panic(err)
				}
			}
			//}
			return nativeImplResult
		}
		return fn
	})
}

func doNativeMethod(arg ProxyArg, nativeImplFunc reflect.Value, session Session) []reflect.Value {
	defer func() {
		err := recover()
		if err != nil {
			var err = session.Rollback()
			if err != nil {
				panic(err)
			}
		}
	}()
	return nativeImplFunc.Call(arg.Args)

}

func haveRollBackType(v []reflect.Value, typeString string) bool {
	//println(typeString)
	if v == nil || len(v) == 0 || typeString == "" {
		return false
	}
	for _, item := range v {
		if item.Kind() == reflect.Interface {
			//println(typeString+" == " + item.String())
			if strings.Contains(item.String(), typeString) {
				if !item.IsNil() {
					return true
				}
			}
		}
	}
	return false
}