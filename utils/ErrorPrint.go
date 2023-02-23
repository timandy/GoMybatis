package utils

import (
	"fmt"

	"github.com/timandy/routine"
)

const PackageName = "com.github.timandy.GoMybatis"

func NewError(StructName string, args ...interface{}) error {
	return routine.NewRuntimeErrorWithMessage(fmt.Sprint("[GoMybatis] ", PackageName, ".", StructName, ": ", args))
}
