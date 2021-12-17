package GoFastExpress

import (
	"fmt"
	"reflect"
)

func GetDeepPtr(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface {
		return v
	}
	if v.IsValid() {
		v = v.Elem()
		if v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
			GetDeepPtr(v)
		}
	}
	return v
}

func GetDeepValue(av reflect.Value, arg interface{}) (interface{}, reflect.Value) {
	if av.Kind() != reflect.Ptr {
		return arg, av
	}
	av = GetDeepPtr(av)
	if av.IsValid() && av.CanInterface() {
		return av.Interface(), av
	}
	return arg, av
}

func isNumberType(t reflect.Type) bool {
	if t != nil {
		switch t.Kind() {
		case reflect.Float32, reflect.Float64:
			fallthrough
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fallthrough
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return true
		}
	}
	return false
}

func toNumberType(v reflect.Value) float64 {
	r, ok := castType(v)
	if ok {
		return r
	}
	var vValue interface{}
	if v.IsValid() && v.CanInterface() {
		vValue = v.Interface()
	}
	panic(fmt.Sprint("[GoFastExpress] cannot convert ", vValue, " (type "+v.Type().String()+") to type float64"))
}

func castType(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true // TODO: Check if uint64 fits into float64.
	}
	return 0, false
}
