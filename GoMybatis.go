package GoMybatis

import (
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/timandy/GoMybatis/v7/ast"
	"github.com/timandy/GoMybatis/v7/lib/github.com/beevik/etree"
	"github.com/timandy/GoMybatis/v7/plugin/page"
	"github.com/timandy/GoMybatis/v7/stmt"
	"github.com/timandy/GoMybatis/v7/utils"
)

const NewSessionFunc = "NewSession" //NewSession method,auto write implement body code

type Mapper struct {
	xml   *etree.Element
	nodes []ast.Node
}

//推荐默认使用单例传入
//根据sessionEngine写入到mapperPtr，value:指向mapper指针反射对象，xml：xml数据，sessionEngine：session引擎，enableLog:是否允许日志输出，log：日志实现
func WriteMapperByValue(value reflect.Value, xml []byte, sessionEngine SessionEngine) {
	if value.Kind() != reflect.Ptr {
		panic("AopProxy: AopProxy arg must be a pointer")
	}
	WriteMapper(value, xml, sessionEngine)
}

//推荐默认使用单例传入
//根据sessionEngine写入到mapperPtr，ptr:指向mapper指针，xml：xml数据，sessionEngine：session引擎，enableLog:是否允许日志输出，log：日志实现
func WriteMapperPtrByEngine(ptr interface{}, xml []byte, sessionEngine SessionEngine) {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr {
		panic("AopProxy: AopProxy arg must be a pointer")
	}
	WriteMapperByValue(v, xml, sessionEngine)
}

//写入方法内容，例如
//type ExampleActivityMapperImpl struct {
//	SelectAll         func(result *[]Activity) error
//	SelectByCondition func(name string, startTime time.Time, endTime time.Time, page int, size int, result *[]Activity) error `args:"name,startTime,endTime,page,size"`
//	UpdateById        func(session *GoMybatis.Session, arg Activity, result *int64) error                                     //只要参数中包含有*GoMybatis.Session的类型，框架默认使用传入的session对象，用于自定义事务
//	Insert            func(arg Activity, result *int64) error
//	CountByCondition  func(name string, startTime time.Time, endTime time.Time, result *int) error `args:"name,startTime,endTime"`
//}
//func的基本类型的参数（例如string,int,time.Time,int64,float....）个数无限制(并且需要用Tag指定参数名逗号隔开,例如`args:"id,phone"`)，返回值必须有error
//func的结构体参数无需指定args的tag，框架会自动扫描它的属性，封装为map处理掉
//使用WriteMapper函数设置代理后即可正常使用。
func WriteMapper(bean reflect.Value, xml []byte, sessionEngine SessionEngine) {
	beanCheck(bean, sessionEngine)
	var mapperTree = LoadMapperXml(xml)
	var decodeErr = sessionEngine.TemplateDecoder().DecodeTree(mapperTree, bean.Type())
	if decodeErr != nil {
		panic(decodeErr)
	}
	//构建期使用的map，无需考虑并发安全
	var methodXmlMap = makeMethodXmlMap(bean, mapperTree, sessionEngine)
	mapperCheck(methodXmlMap)
	var resultMaps = makeResultMaps(mapperTree)
	mapperResultMapCheck(resultMaps)
	var returnTypeMap = makeReturnTypeMap(bean.Elem().Type())
	var beanName = bean.Type().PkgPath() + bean.Type().String()

	ProxyValue(bean, func(funcField reflect.StructField, field reflect.Value) func(arg ProxyArg) []reflect.Value {
		//构建期
		var funcName = funcField.Name
		var returnType = returnTypeMap[funcName]
		if returnType == nil {
			returnType = returnVoid
		}
		//mapper
		var mapper = methodXmlMap[funcName]
		//resultMaps
		var resultMap map[string]*ResultProperty

		if funcName != NewSessionFunc {
			var resultMapId = mapper.xml.SelectAttrValue(Element_ResultMap, "")
			if resultMapId != "" {
				resultMap = resultMaps[resultMapId]
			}
		}

		//执行期
		if funcName == NewSessionFunc {
			var proxyFunc = func(arg ProxyArg) []reflect.Value {
				var returnValue *reflect.Value = nil
				//build return Type
				if returnType.ReturnOutType != nil {
					var returnV = reflect.New(*returnType.ReturnOutType)
					switch (*returnType.ReturnOutType).Kind() {
					case reflect.Map:
						returnV.Elem().Set(reflect.MakeMap(*returnType.ReturnOutType))
					case reflect.Slice:
						returnV.Elem().Set(reflect.MakeSlice(*returnType.ReturnOutType, 0, 0))
					}
					returnValue = &returnV
				}
				var session = sessionEngine.SessionFactory().NewSession(beanName, SessionType_Default)
				returnValue.Elem().Set(reflect.ValueOf(session).Elem().Addr().Convert(*returnType.ReturnOutType))
				return buildReturnValues(returnType, returnValue, nil)
			}
			return proxyFunc
		} else {
			var proxyFunc = func(arg ProxyArg) []reflect.Value {
				var returnValue *reflect.Value = nil
				//build return Type
				if returnType.ReturnOutType != nil {
					var returnV = reflect.New(*returnType.ReturnOutType)
					switch (*returnType.ReturnOutType).Kind() {
					case reflect.Map:
						returnV.Elem().Set(reflect.MakeMap(*returnType.ReturnOutType))
					case reflect.Slice:
						returnV.Elem().Set(reflect.MakeSlice(*returnType.ReturnOutType, 0, 0))
					case reflect.Pointer:
						elemType := (*returnType.ReturnOutType).Elem()
						returnV.Elem().Set(reflect.New(elemType))
					}
					returnValue = &returnV
				}
				//exe sql
				autoFieldValue, err := exeMethodByXml(mapper.xml.Tag, beanName, sessionEngine, arg, mapper.nodes, resultMap, returnValue)
				if err != nil && sessionEngine.IsPanicOnError() {
					panic(err)
				}
				if sessionEngine.IsWriteBackAutoFiled() {
					writeBackAutoField(arg, returnType.AutoFiledName, autoFieldValue)
				}
				return buildReturnValues(returnType, returnValue, err)
			}
			return proxyFunc
		}
	})
}

func writeBackAutoField(arg ProxyArg, fieldName string, fieldValue int64) {
	if fieldValue == -1 || len(fieldName) == 0 || len(arg.Args) != 1 {
		return
	}

	value := arg.Args[0]
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return
	}

	value.FieldByName(fieldName).SetInt(fieldValue)
}

func mapperCheck(arg map[string]*Mapper) {
	//TODO check mapper
}

func mapperResultMapCheck(arg map[string]map[string]*ResultProperty) {
	if arg == nil {
		return
	}
	for resultMap, item := range arg {
		if item == nil {
			return
		}
		for k, v := range item {
			if v.Column == "" {
				panic("[GoMybatis] in mapper .resultMap: " + resultMap + "." + k + " 'column' can not be empty!")
			}
			if v.LangType == "" {
				panic("[GoMybatis] in mapper .resultMap: " + resultMap + "." + k + " 'langType' can not be empty!")
			}
		}
	}
}

//check beans
func beanCheck(value reflect.Value, engine SessionEngine) {
	var t = value.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		var fieldItem = t.Field(i)
		if fieldItem.Type.Kind() != reflect.Func {
			continue
		}
		var argsLen = fieldItem.Type.NumIn() //参数长度，除session参数外。
		var customLen = 0
		for argIndex := 0; argIndex < fieldItem.Type.NumIn(); argIndex++ {
			var inType = fieldItem.Type.In(argIndex)
			if isCustomStruct(inType) {
				customLen++
			}
		}
		if argsLen > 1 && customLen > 1 {
			if engine.LogEnable() {
				engine.Log().Println("[GoMybatis] %v has more than one struct parameters", fieldItem.Name)
			}
		}
	}
}

func buildReturnValues(returnType *ReturnType, returnValue *reflect.Value, e error) []reflect.Value {
	var returnValues = make([]reflect.Value, returnType.NumOut)
	for index, _ := range returnValues {
		if index == returnType.ReturnIndex {
			if returnValue != nil {
				returnValues[index] = (*returnValue).Elem()
			}
		} else {
			if e != nil {
				returnValues[index] = reflect.New(*returnType.ErrorType)
				returnValues[index].Elem().Set(reflect.ValueOf(e))
				returnValues[index] = returnValues[index].Elem()
			} else {
				returnValues[index] = reflect.Zero(*returnType.ErrorType)
			}
		}
	}
	return returnValues
}

func makeReturnTypeMap(value reflect.Type) (returnMap map[string]*ReturnType) {
	returnMap = make(map[string]*ReturnType)
	var proxyType = value
	for i := 0; i < proxyType.NumField(); i++ {
		var funcType = proxyType.Field(i).Type
		var funcName = proxyType.Field(i).Name

		if funcType.Kind() != reflect.Func {
			if funcType.Kind() == reflect.Struct {
				var childMap = makeReturnTypeMap(funcType)
				for k, v := range childMap {
					returnMap[k] = v
				}
			}
			continue
		}

		var numOut = funcType.NumOut()
		if numOut > 2 {
			panic("[GoMybatis] func '" + funcName + "()' return num out must in [0-2]!")
		}
		for f := 0; f < numOut; f++ {
			var outType = funcType.Out(f)
			if funcName != NewSessionFunc {
				//过滤NewSession方法
				if outType.Kind() == reflect.Interface && outType.String() != "error" {
					panic("[GoMybatis] func '" + funcName + "()' return '" + outType.String() + "' can not be a 'interface'!")
				}
			}
			var returnType = returnMap[funcName]
			if returnType == nil {
				returnMap[funcName] = &ReturnType{
					ReturnIndex: -1,
					NumOut:      numOut,
				}
			}
			if outType.String() != "error" {
				returnMap[funcName].ReturnIndex = f
				returnMap[funcName].ReturnOutType = &outType
			} else {
				//error
				returnMap[funcName].ErrorType = &outType
			}
		}

		if funcType.NumIn() == 1 && isCustomStruct(funcType.In(0)) {
			var returnType = returnMap[funcName]
			if returnType == nil {
				returnMap[funcName] = &ReturnType{
					ReturnIndex: -1,
					NumOut:      numOut,
				}
			}
			inType := funcType.In(0)
			if inType.Kind() == reflect.Ptr {
				inType = inType.Elem()
			}
			if inType.Kind() == reflect.Struct {
				for i := 0; i < inType.NumField(); i++ {
					structField := inType.Field(i)
					if structField.Tag.Get("type") == "auto" {
						returnMap[funcName].AutoFiledName = structField.Name
						break
					}
				}
			}
		}
	}
	return returnMap
}

//map[id]map[cloum]Property
func makeResultMaps(xmls map[string]etree.Token) map[string]map[string]*ResultProperty {
	var resultMaps = make(map[string]map[string]*ResultProperty)
	for _, item := range xmls {
		var typeString = reflect.TypeOf(item).String()
		if typeString == "*etree.Element" {
			var xmlItem = item.(*etree.Element)
			if xmlItem.Tag == Element_ResultMap {
				var resultPropertyMap = make(map[string]*ResultProperty)
				for _, elementItem := range xmlItem.ChildElements() {
					var property = ResultProperty{
						XMLName:  elementItem.Tag,
						Column:   elementItem.SelectAttrValue("column", ""),
						LangType: elementItem.SelectAttrValue("langType", ""),
					}
					resultPropertyMap[property.Column] = &property
				}
				resultMaps[xmlItem.SelectAttrValue("id", "")] = resultPropertyMap
			}
		}
	}
	return resultMaps
}

//return a map map[`method`]*MapperXml
func makeMethodXmlMap(bean reflect.Value, mapperTree map[string]etree.Token, engine SessionEngine) map[string]*Mapper {
	var beanType = bean.Type()
	if beanType.Kind() == reflect.Ptr {
		beanType = beanType.Elem()
	}

	var methodXmlMap = make(map[string]*Mapper)
	var totalField = beanType.NumField()
	for i := 0; i < totalField; i++ {
		var fieldItem = beanType.Field(i)
		if fieldItem.Type.Kind() == reflect.Func {
			//field must be func
			methodFieldCheck(&beanType, &fieldItem, engine.IsPrintWarning())
			var mapperXml = findMapperXml(mapperTree, fieldItem.Name)
			if mapperXml != nil {
				methodXmlMap[fieldItem.Name] = &Mapper{
					xml:   mapperXml,
					nodes: engine.SqlBuilder().NodeParser().Parser(mapperXml.Child),
				}
			} else {
				if fieldItem.Name == NewSessionFunc {
					//过滤NewSession方法
					continue
				}
				panic("[GoMybatis] can not find method " + beanType.String() + "." + fieldItem.Name + "() in xml !")
			}
		}
	}
	return methodXmlMap
}

//方法基本规则检查
func methodFieldCheck(beanType *reflect.Type, methodType *reflect.StructField, warning bool) {
	var args = methodType.Tag.Get("args")
	if methodType.Type.NumOut() > 1 && args == "" && !(methodType.Name == "NewSession") {
		if warning {
			log.Println("[GoMybatis] warning ======================== " + (*beanType).Name() + "." + methodType.Name + "() have not define tag args:\"\",maybe can not get param value!")
		}
	}
}

func findMapperXml(mapperTree map[string]etree.Token, methodName string) *etree.Element {
	for _, mapperXml := range mapperTree {
		//exec sql,return data
		var typeString = reflect.TypeOf(mapperXml).String()
		if typeString == "*etree.Element" {
			var key = mapperXml.(*etree.Element).SelectAttrValue("id", "")
			if strings.EqualFold(key, methodName) {
				return mapperXml.(*etree.Element)
			}
		}
	}
	return nil
}

func exeMethodByXml(elementType ElementType, beanName string, sessionEngine SessionEngine, proxyArg ProxyArg, nodes []ast.Node, resultMap map[string]*ResultProperty, returnValue *reflect.Value) (int64, error) {
	session, err := findArgSession(proxyArg)
	if err != nil {
		return -1, err
	}
	if session == nil {
		var goroutineID int64 //协程id
		if sessionEngine.GoroutineIDEnable() {
			goroutineID = utils.GoroutineID()
		} else {
			goroutineID = 0
		}
		session = sessionEngine.GoroutineSessionMap().Get(goroutineID)
	}
	if session == nil {
		s, err := sessionEngine.NewSession(beanName)
		if err != nil {
			return -1, err
		}
		session = s
		defer session.Close()
	}
	convert, err := session.StmtConvert()
	if err != nil {
		return -1, err
	}
	array_arg := []interface{}{}
	sql, pageArg, err := buildSql(proxyArg, nodes, sessionEngine.SqlBuilder(), &array_arg, convert)
	if err != nil {
		return -1, err
	}

	//do CRUD
	haveLastReturnValue := returnValue != nil && (*returnValue).IsNil() == false
	if elementType == Element_Select && haveLastReturnValue {
		pageResult, isPageResult := returnValue.Interface().(page.IPageResult)
		if !isPageResult && returnValue.Kind() == reflect.Ptr {
			pageResult, isPageResult = returnValue.Elem().Interface().(page.IPageResult)
		}
		if pageArg != nil && isPageResult {
			//分页查询
			page.Assert(pageArg)
			//1. 执行count
			countSql := "select count(*) ROW_COUNT from (" + sql + ") _____tmp_____"
			if sessionEngine.LogEnable() {
				sessionEngine.Log().Println("[GoMybatis] [%v] Query ==> %v", session.Id(), countSql)
				sessionEngine.Log().Println("[GoMybatis] [%v] Args  ==> %v", session.Id(), utils.SprintArray(array_arg))
			}
			countRes, countErr := session.QueryPrepare(countSql, array_arg...)
			if sessionEngine.LogEnable() {
				var RowsAffected = "0"
				if countErr == nil && countRes != nil {
					RowsAffected = strconv.Itoa(len(countRes))
				}
				sessionEngine.Log().Println("[GoMybatis] [%v] ReturnRows <== %v", session.Id(), RowsAffected)
				if countErr != nil {
					sessionEngine.Log().Println("[GoMybatis] [%v] error == %v", session.Id(), countErr.Error())
				}
			}
			if countErr != nil {
				return -1, countErr
			}
			rowCount := resolveCountRes(countRes)
			//2. 执行 offset limit
			querySql := sql + " limit " + strconv.Itoa(page.GetLimit(pageArg)) + " offset " + strconv.Itoa(page.GetOffset(pageArg))
			if sessionEngine.LogEnable() {
				sessionEngine.Log().Println("[GoMybatis] [%v] Query ==> %v", session.Id(), querySql)
				sessionEngine.Log().Println("[GoMybatis] [%v] Args  ==> %v", session.Id(), utils.SprintArray(array_arg))
			}
			queryRes, queryErr := session.QueryPrepare(querySql, array_arg...)
			if sessionEngine.LogEnable() {
				var RowsAffected = "0"
				if queryErr == nil && queryRes != nil {
					RowsAffected = strconv.Itoa(len(queryRes))
				}
				sessionEngine.Log().Println("[GoMybatis] [%v] ReturnRows <== %v", session.Id(), RowsAffected)
				if queryErr != nil {
					sessionEngine.Log().Println("[GoMybatis] [%v] error == %v", session.Id(), queryErr.Error())
				}
			}
			if queryErr != nil {
				return -1, queryErr
			}
			//处理返回结果,解析分页返回值的 SetList 方法
			setListMethod, setListMethodArgValue := resolveSetListMethod(returnValue)
			if err := sessionEngine.SqlResultDecoder().Decode(resultMap, queryRes, setListMethodArgValue.Interface()); err != nil {
				return -1, err
			}
			//为分页返回值字段填充值
			pageResult.SetTotalCount(int64(rowCount))
			pageResult.SetPageCount(int(math.Ceil(float64(rowCount) / float64(pageArg.GetPageSize()))))
			pageResult.SetDisplayCount(len(queryRes))
			setListMethod.Call([]reflect.Value{reflect.Indirect(setListMethodArgValue)})
			return -1, nil
		}

		//非分页查询
		if sessionEngine.LogEnable() {
			sessionEngine.Log().Println("[GoMybatis] [%v] Query ==> %v", session.Id(), sql)
			sessionEngine.Log().Println("[GoMybatis] [%v] Args  ==> %v", session.Id(), utils.SprintArray(array_arg))
		}
		res, err := session.QueryPrepare(sql, array_arg...)
		if sessionEngine.LogEnable() {
			var RowsAffected = "0"
			if err == nil && res != nil {
				RowsAffected = strconv.Itoa(len(res))
			}
			sessionEngine.Log().Println("[GoMybatis] [%v] ReturnRows <== %v", session.Id(), RowsAffected)
			if err != nil {
				sessionEngine.Log().Println("[GoMybatis] [%v] error == %v", session.Id(), err.Error())
			}
		}
		if err != nil {
			return -1, err
		}

		if err := sessionEngine.SqlResultDecoder().Decode(resultMap, res, returnValue.Interface()); err != nil {
			return -1, err
		}
		return -1, err
	}
	if sessionEngine.LogEnable() {
		sessionEngine.Log().Println("[GoMybatis] [%v] Exec ==> %v", session.Id(), sql)
		sessionEngine.Log().Println("[GoMybatis] [%v] Args ==> %v", session.Id(), utils.SprintArray(array_arg))
	}
	res, err := session.ExecPrepare(sql, array_arg...)
	if sessionEngine.LogEnable() {
		var RowsAffected = "0"
		if err == nil && res != nil {
			RowsAffected = strconv.FormatInt(res.RowsAffected, 10)
		}
		sessionEngine.Log().Println("[GoMybatis] [%v] RowsAffected <== %v", session.Id(), RowsAffected)
		if err != nil {
			sessionEngine.Log().Println("[GoMybatis] [%v] error == %v", session.Id(), err.Error())
		}
	}

	if err != nil {
		return -1, err
	}
	if haveLastReturnValue {
		returnValue.Elem().SetInt(res.RowsAffected)
	}
	return res.LastInsertId, err
}

// 根据返回值类型, 创建 PageResult.SetList(x) 方法的第一个参数的实例, 并返回反射方法
func resolveSetListMethod(returnValue *reflect.Value) (setListMethod reflect.Value, setListMethodArgValue reflect.Value) {
	pageResultStructInstance := *returnValue
	setListMethod = pageResultStructInstance.MethodByName("SetList")
	if (!setListMethod.IsValid() || setListMethod.Kind() != reflect.Func) && pageResultStructInstance.Kind() == reflect.Pointer {
		setListMethod = pageResultStructInstance.Elem().MethodByName("SetList")
	}
	if !setListMethod.IsValid() || setListMethod.Kind() != reflect.Func {
		panic("Your pageResult type must have method SetList([]T)")
	}
	setListMethodType := setListMethod.Type()
	setListMethodArgCount := setListMethodType.NumIn()
	if setListMethodArgCount != 1 {
		panic("Your SetList([]T) method of pageResult type must have only one argument")
	}
	setListMethodFirstArgType := setListMethodType.In(0)
	setListMethodArgValue = reflect.New(setListMethodFirstArgType) //构建 PageResult.List 类型的实例
	setListMethodArgValue.Elem().Set(reflect.MakeSlice(setListMethodFirstArgType, 0, 0))
	return
}

//解析分页统计行数执行结果
func resolveCountRes(res []map[string][]byte) int {
	if len(res) != 1 {
		panic("count sql returned rows is not 1")
	}
	mp := res[0]
	buf, exits := mp["ROW_COUNT"]
	if !exits {
		panic("no column ROW_COUNT returned")
	}
	rowCount, err := strconv.Atoi(string(buf))
	if err != nil {
		panic(err)
	}
	return rowCount
}

func closeSession(factory *SessionFactory, session Session) {
	if session == nil {
		return
	}
	factory.Close(session.Id())
	session.Close()
}

func findArgSession(proxyArg ProxyArg) (Session, error) {
	var session Session
	for _, arg := range proxyArg.Args {
		var argInterface = arg.Interface()
		if arg.Kind() == reflect.Ptr &&
			arg.IsNil() == false &&
			argInterface != nil &&
			arg.Type().String() == GoMybatis_Session_Ptr {
			session = *(argInterface.(*Session))
			continue
		} else if argInterface != nil &&
			arg.Kind() == reflect.Interface &&
			arg.Type().String() == GoMybatis_Session {
			session = argInterface.(Session)
			continue
		}
	}
	return session, nil
}

//解析语法树, 构建 sql
func buildSql(proxyArg ProxyArg, nodes []ast.Node, sqlBuilder SqlBuilder, array_arg *[]interface{}, stmtConvert stmt.StmtIndexConvert) (string, page.IPageArg, error) {
	var paramMap = make(map[string]interface{})
	var tagArgsLen = proxyArg.TagArgsLen
	var argsLen = proxyArg.ArgsLen //参数长度，除session参数外。
	var customLen = 0
	var customIndex = -1
	var pageArg page.IPageArg
	for argIndex, arg := range proxyArg.Args {
		var argInterface = arg.Interface()
		//分页参数, 要求实现 IPageArg 时接收器不能使用指针
		if pa, ok := argInterface.(page.IPageArg); ok {
			pageArg = pa
			//分页参数中可能包含其他业务参数, 需要继续解析
		}
		if arg.Kind() == reflect.Ptr &&
			arg.IsNil() == false &&
			argInterface != nil &&
			arg.Type().String() == GoMybatis_Session_Ptr {
			continue
		} else if argInterface != nil &&
			arg.Kind() == reflect.Interface &&
			arg.Type().String() == GoMybatis_Session {
			continue
		}
		if isCustomStruct(arg.Type()) {
			customLen++
			customIndex = argIndex
		}
		if arg.Type().String() == GoMybatis_Session_Ptr ||
			arg.Type().String() == GoMybatis_Session {
			if argsLen > 0 {
				argsLen--
			}
			if tagArgsLen > 0 {
				tagArgsLen--
			}
		}
		if tagArgsLen > 0 && argIndex < tagArgsLen &&
			proxyArg.TagArgs[argIndex].Name != "" {
			//插入2份参数，兼容大小写不敏感的参数
			var lowerKey = utils.LowerFieldFirstName(proxyArg.TagArgs[argIndex].Name)
			var upperKey = utils.UpperFieldFirstName(proxyArg.TagArgs[argIndex].Name)
			paramMap[lowerKey] = argInterface
			paramMap[upperKey] = argInterface
		} else {
			//未命名参数，为arg加参数位置，例如 arg0,arg1,arg2....
			paramMap[DefaultOneArg+strconv.Itoa(argIndex)] = argInterface
		}
	}
	if customLen == 1 && customIndex != -1 {
		//只有一个结构体参数，需要展开它的成员变量 加入到map
		var tag *TagArg
		if proxyArg.TagArgsLen == 1 {
			tag = &proxyArg.TagArgs[0]
		}
		expandStructToMap := scanStructArgFields(proxyArg.Args[customIndex], tag)
		for key, value := range expandStructToMap {
			paramMap[key] = value
		}
	}
	result, err := sqlBuilder.BuildSql(paramMap, nodes, array_arg, stmtConvert)
	return result, pageArg, err
}

//scan params
func scanStructArgFields(v reflect.Value, tag *TagArg) map[string]interface{} {
	var t = v.Type()
	parameters := make(map[string]interface{})
	if v.Kind() == reflect.Ptr {
		if v.IsNil() == true {
			return parameters
		}
		//为指针，解引用
		v = v.Elem()
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(`[GoMybatis] the scanParamterBean() arg is not a struct type!,type =` + t.String())
	}

	var structArg = make(map[string]interface{})

	//json arg,性能较差
	//var vptr=v.Interface()
	//var js,_=json.Marshal(vptr)
	//json.Unmarshal(js,&structArg)
	//
	//for key,value:=range structArg {
	//	parameters[key]=value
	//}

	//reflect arg,性能较快
	for i := 0; i < t.NumField(); i++ {
		var typeValue = t.Field(i)
		var field = v.Field(i)

		var obj interface{}
		if field.CanInterface() {
			obj = field.Interface()
		}
		var jsonKey = typeValue.Tag.Get(`json`)
		if strings.Index(jsonKey, ",") != -1 {
			jsonKey = strings.Split(jsonKey, ",")[0]
		}
		if jsonKey != "" {
			parameters[jsonKey] = obj
			structArg[jsonKey] = obj
			parameters[typeValue.Name] = obj
			structArg[typeValue.Name] = obj
		} else {
			parameters[typeValue.Name] = obj
			structArg[typeValue.Name] = obj
		}
	}
	if tag != nil && parameters[tag.Name] == nil {
		parameters[tag.Name] = structArg
	}
	return parameters
}

func isCustomStruct(value reflect.Type) bool {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	return value.Kind() == reflect.Struct &&
		value.String() != GoMybatis_Time &&
		value.String() != GoMybatis_Time_Ptr
}
