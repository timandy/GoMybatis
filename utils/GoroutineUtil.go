package utils

import "github.com/timandy/routine"

func GoroutineID() int64 {
	return routine.Goid()
}
