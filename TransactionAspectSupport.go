package GoMybatis

import (
	"fmt"
	"github.com/timandy/GoMybatis/v7/tx"
	"github.com/timandy/GoMybatis/v7/utils"
	"reflect"
	"strings"
)

//使用AOP切面 代理目标服务，如果服务painc()它的事务会回滚
//默认为单协程模型，如果是多协程调用的情况请开启engine.SetGoroutineIDEnable(true)
func AopProxyService(service interface{}, engine SessionEngine) {
	var v = reflect.ValueOf(service)
	if v.Kind() != reflect.Ptr {
		panic("[GoMybatis] AopProxy service  must use ptr arg!")
	}
	AopProxyServiceValue(v, engine)
}

//使用AOP切面 代理目标服务，如果服务painc()它的事务会回滚
func AopProxyServiceValue(service reflect.Value, engine SessionEngine) {
	var beanType = service.Type().Elem()
	var beanName = beanType.PkgPath() + beanType.Name()
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
			var goroutineID int64 //协程id
			if engine.GoroutineIDEnable() {
				goroutineID = utils.GoroutineID()
			} else {
				goroutineID = 0
			}
			var session = engine.GoroutineSessionMap().Get(goroutineID)
			if session == nil {
				//todo newSession is use service bean name?
				var err error
				session, err = engine.NewSession(beanName)
				defer func() {
					if session != nil {
						session.Close()
					}
					engine.GoroutineSessionMap().Delete(goroutineID)
				}()
				if err != nil {
					panic(err)
				}
				//压入map
				engine.GoroutineSessionMap().Put(goroutineID, session)
			}
			if !haveTx {
				var err = session.Begin(session.LastPROPAGATION())
				if err != nil {
					panic(err)
				}
			} else {
				var err = session.Begin(&propagation)
				if err != nil {
					panic(err)
				}
			}

			var nativeImplResult = doNativeMethod(funcField, arg, nativeImplFunc, session, engine.Log())
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
			return nativeImplResult
		}
		return fn
	})
}

func doNativeMethod(funcField reflect.StructField, arg ProxyArg, nativeImplFunc reflect.Value, session Session, log Log) []reflect.Value {
	defer func() {
		err := recover()
		if err != nil {
			var rollbackErr = session.Rollback()
			if rollbackErr != nil {
				panic(fmt.Sprint(err) + rollbackErr.Error())
			}
			if log != nil {
				log.Println((fmt.Sprint(err) + " Throw out error will Rollback! from >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> " + funcField.Name + "()"))
			}
			panic(err)
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
