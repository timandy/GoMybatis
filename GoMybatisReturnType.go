package GoMybatis

import (
	"reflect"
)

type ReturnType struct {
	ErrorType     *reflect.Type
	ReturnOutType *reflect.Type
	AutoFiledName string //自增字段名 type:"auto"
	ReturnIndex   int    //返回数据位置索引
	NumOut        int    //返回总数
}

var returnVoid = &ReturnType{
	ErrorType:     nil,
	ReturnOutType: nil,
	AutoFiledName: "",
	ReturnIndex:   -1,
	NumOut:        0,
}
