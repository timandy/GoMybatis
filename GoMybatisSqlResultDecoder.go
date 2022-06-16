package GoMybatis

import (
	"encoding/json"
	"github.com/timandy/GoMybatis/v7/utils"
	"reflect"
	"strings"
	"unicode"
)

type GoMybatisSqlResultDecoder struct {
	SqlResultDecoder
}

func (it GoMybatisSqlResultDecoder) Decode(resultMap map[string]*ResultProperty, sqlResult []map[string][]byte, result interface{}) error {
	if sqlResult == nil || result == nil {
		return nil
	}
	var resultV = reflect.ValueOf(result)
	if resultV.Kind() == reflect.Ptr {
		resultV = resultV.Elem()
	} else {
		panic("[GoMybatis] SqlResultDecoder only support ptr type,make sure use '*Your Type'!")
	}

	var value = []byte{}
	var sqlResultLen = len(sqlResult)
	if sqlResultLen == 0 {
		return nil
	}
	if !isArray(resultV) {
		//single basic type
		if sqlResultLen > 1 {
			return utils.NewError("SqlResultDecoder", " Decode one result,but find database result size find > 1 !")
		}
		// base type convert
		if isBasicType(resultV.Type()) {
			basicTypeName := resultV.Type().Name()
			for _, s := range sqlResult[0] {
				var b = strings.Builder{}
				if isStringInJson(basicTypeName) || (resultV.Kind() == reflect.Struct) {
					b.WriteString("\"")
					b.Write(s)
					b.WriteString("\"")
				} else if isBoolInJson(basicTypeName) {
					b.Write(toBoolString(s))
				} else {
					b.Write(s)
				}
				value = []byte(b.String())
				break
			}
		} else {
			var structMap, basicType, e = makeStructMap(resultV.Type())
			if e != nil {
				return e
			}
			value = makeJsonObjBytes(resultMap, sqlResult[0], structMap, basicType)
		}
	} else {
		if resultV.Type().Kind() != reflect.Array && resultV.Type().Kind() != reflect.Slice {
			return utils.NewError("SqlResultDecoder", " decode type not an struct array or slice!")
		}
		var resultVItemType = resultV.Type().Elem()
		var structMap, basicType, e = makeStructMap(resultVItemType)
		if e != nil {
			return e
		}
		var done = len(sqlResult) - 1
		var index = 0
		var jsonData = strings.Builder{}
		jsonData.WriteString("[")
		for _, v := range sqlResult {
			jsonData.Write(makeJsonObjBytes(resultMap, v, structMap, basicType))
			//write ','
			if index < done {
				jsonData.WriteString(",")
			}
			index += 1
		}
		jsonData.WriteString("]")
		value = []byte(jsonData.String())
	}
	e := json.Unmarshal(value, result)
	return e
}

func makeStructMap(itemType reflect.Type) (structMap map[string]*reflect.Type, basicType *reflect.Type, err error) {
	structMap = map[string]*reflect.Type{}
	basicType = makeStructMapCore(itemType, structMap)
	return
}

func makeStructMapCore(itemType reflect.Type, structMap map[string]*reflect.Type) *reflect.Type {
	if itemType.Kind() == reflect.Ptr {
		itemType = itemType.Elem()
	}

	switch itemType.Kind() {
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String:
		return &itemType

	case reflect.Struct:
		if itemType.String() == "time.Time" {
			return &itemType
		}

		for i := 0; i < itemType.NumField(); i++ {
			structField := itemType.Field(i)
			if structField.Anonymous {
				makeStructMapCore(structField.Type, structMap)
				continue
			}
			jsonTag := structField.Tag.Get("json")
			if len(jsonTag) == 0 {
				structMap[structField.Name] = &structField.Type
			} else {
				structMap[jsonTag] = &structField.Type
			}
		}
		return nil

	default:
		return nil
	}
}

//字符串转换为大写驼峰样式, 例如 hello_world => HelloWorld
func toUpperCamelCase(filedName string) string {
	strLen := len(filedName)
	if strLen == 0 {
		return ""
	}

	builder := &strings.Builder{}
	findUnderLine := true
	for _, c := range filedName {
		if c == '_' {
			findUnderLine = true
			continue
		}

		if findUnderLine {
			builder.WriteRune(unicode.ToUpper(c))
			findUnderLine = false
			continue
		}

		builder.WriteRune(c)
	}
	return builder.String()
}

func isStringInJson(typeName string) bool {
	return typeName == "string" ||
		typeName == "*string" ||
		typeName == "time.Time" ||
		typeName == "*time.Time"
}

func isBoolInJson(typeName string) bool {
	return typeName == "bool" ||
		typeName == "*bool"
}

func toBoolString(bytes []byte) []byte {
	if len(bytes) == 1 && bytes[0] == '0' {
		return []byte("false")
	}
	return []byte("true")
}

//make an json value
func makeJsonObjBytes(resultMap map[string]*ResultProperty, sqlData map[string][]byte, structMap map[string]*reflect.Type, basicType *reflect.Type) []byte {
	if len(sqlData) == 1 && basicType != nil {
		basicTypeName := (*basicType).String()
		isString := isStringInJson(basicTypeName)
		isBool := isBoolInJson(basicTypeName)
		for _, sqlV := range sqlData { //只有一列,所以只遍历一次
			if len(sqlV) == 0 {
				return []byte("null")
			} else if isString {
				return []byte("\"" + encodeStringValue(sqlV) + "\"")
			} else if isBool {
				return toBoolString(sqlV)
			} else {
				return sqlV
			}
		}
		//make sure return
		return nil
	}

	var jsonData = strings.Builder{}
	jsonData.WriteString("{")

	var done = len(sqlData) - 1
	var index = 0
	for k, sqlV := range sqlData {
		//字段名
		fieldName := k
		if _, exists := structMap[k]; !exists {
			fieldName = toUpperCamelCase(k)
		}
		jsonData.WriteString("\"")
		jsonData.WriteString(fieldName)
		jsonData.WriteString("\":")

		var isStringType = false
		var isBoolType = false
		var fetched = true
		if resultMap != nil {
			var resultMapItem = resultMap[k]
			if resultMapItem == nil {
				fetched = false
			} else {
				if isStringInJson(resultMapItem.LangType) {
					isStringType = true
				} else if isBoolInJson(resultMapItem.LangType) {
					isBoolType = true
				}
			}
		} else if structMap != nil {
			var v = structMap[fieldName]
			if v == nil {
				fetched = false
			} else {
				if isStringInJson((*v).String()) {
					isStringType = true
				} else if isBoolInJson((*v).String()) {
					isBoolType = true
				}
			}
		} else {
			isStringType = true
		}
		if fetched {
			if sqlV == nil || len(sqlV) == 0 {
				sqlV = []byte("null")
				jsonData.Write(sqlV)
			} else if isStringType {
				jsonData.WriteString("\"")
				jsonData.WriteString(encodeStringValue(sqlV))
				jsonData.WriteString("\"")
			} else if isBoolType {
				sqlV = toBoolString(sqlV)
				jsonData.Write(sqlV)
			} else {
				jsonData.Write(sqlV)
			}
		} else {
			sqlV = []byte("null")
			jsonData.Write(sqlV)
		}
		//write ','
		if index < done {
			jsonData.WriteString(",")
		}
		index += 1
	}
	jsonData.WriteString("}")
	return []byte(jsonData.String())
}

func encodeStringValue(v []byte) string {
	if v == nil {
		return "null"
	}
	if len(v) == 0 {
		return ""
	}
	var s = string(v)
	var b, e = json.Marshal(s)
	if e != nil || len(b) == 0 {
		return "null"
	}
	s = string(b[1 : len(b)-1])
	return s
}

// is an array or slice
func isArray(val reflect.Value) bool {
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	kind := typ.Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func isBasicType(tItemTypeFieldType reflect.Type) bool {
	if tItemTypeFieldType.Kind() == reflect.Bool ||
		tItemTypeFieldType.Kind() == reflect.Int ||
		tItemTypeFieldType.Kind() == reflect.Int8 ||
		tItemTypeFieldType.Kind() == reflect.Int16 ||
		tItemTypeFieldType.Kind() == reflect.Int32 ||
		tItemTypeFieldType.Kind() == reflect.Int64 ||
		tItemTypeFieldType.Kind() == reflect.Uint ||
		tItemTypeFieldType.Kind() == reflect.Uint8 ||
		tItemTypeFieldType.Kind() == reflect.Uint16 ||
		tItemTypeFieldType.Kind() == reflect.Uint32 ||
		tItemTypeFieldType.Kind() == reflect.Uint64 ||
		tItemTypeFieldType.Kind() == reflect.Float32 ||
		tItemTypeFieldType.Kind() == reflect.Float64 ||
		tItemTypeFieldType.Kind() == reflect.String {
		return true
	}
	if tItemTypeFieldType.Kind() == reflect.Struct && tItemTypeFieldType.String() == "time.Time" {
		return true
	}
	return false
}
