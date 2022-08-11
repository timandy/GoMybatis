package GoMybatis

import (
	"reflect"

	"github.com/timandy/GoMybatis/v7/lib/github.com/beevik/etree"
)

type TemplateDecoder interface {
	SetPrintElement(print bool)
	DecodeTree(tree map[string]etree.Token, beanType reflect.Type) error
}
