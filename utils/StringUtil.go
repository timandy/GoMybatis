package utils

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

//首字母转大写
func UpperFieldFirstName(fieldStr string) string {
	if fieldStr != "" {
		var fieldBytes = []byte(fieldStr)
		var fieldLength = len(fieldStr)
		fieldStr = strings.ToUpper(string(fieldBytes[:1])) + string(fieldBytes[1:fieldLength])
		fieldBytes = nil
	}
	return fieldStr
}

//首字母转小写
func LowerFieldFirstName(fieldStr string) string {
	if fieldStr != "" {
		var fieldBytes = []byte(fieldStr)
		var fieldLength = len(fieldStr)
		fieldStr = strings.ToLower(string(fieldBytes[:1])) + string(fieldBytes[1:fieldLength])
		fieldBytes = nil
	}
	return fieldStr
}

// format array [1,2,3,""] to '[1,2,3,]'
func SprintArray(array_or_slice []interface{}) string {
	if len(array_or_slice) == 0 {
		return ""
	}

	builder := &strings.Builder{}
	rawValue, isString := GetRawValue(array_or_slice[0])
	if isString {
		builder.WriteRune('"')
		builder.WriteString(fmt.Sprint(rawValue))
		builder.WriteRune('"')
	} else {
		builder.WriteString(fmt.Sprint(rawValue))
	}
	for i := 1; i < len(array_or_slice); i++ {
		builder.WriteString(", ")
		rawValue, isString := GetRawValue(array_or_slice[i])
		if isString {
			builder.WriteRune('"')
			builder.WriteString(fmt.Sprint(rawValue))
			builder.WriteRune('"')
		} else {
			builder.WriteString(fmt.Sprint(rawValue))
		}
	}
	return builder.String()
}

// isValue reports whether v is a valid Value parameter type.
func isValue(v interface{}) (isValid bool, isString bool) {
	if v == nil {
		return true, false
	}
	switch v.(type) {
	case []byte, bool, float64, int64:
		return true, false
	case string, time.Time:
		return true, true
	}
	return false, false
}

func GetRawValue(v interface{}) (rawValue interface{}, isString bool) {
	if isValid, isString := isValue(v); isValid {
		return v, isString
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr:
		// indirect pointers
		if rv.IsNil() {
			return nil, false
		} else {
			return GetRawValue(rv.Elem().Interface())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), false
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(rv.Uint()), false
	case reflect.Uint64:
		u64 := rv.Uint()
		return int64(u64), false
	case reflect.Float32, reflect.Float64:
		return rv.Float(), false
	case reflect.Bool:
		return rv.Bool(), false
	case reflect.Slice:
		ek := rv.Type().Elem().Kind()
		if ek == reflect.Uint8 {
			return rv.Bytes(), false
		}
		return v, false
	case reflect.String:
		return rv.String(), true
	}
	return v, false
}
