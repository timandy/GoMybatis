package GoMybatis

import (
	"bytes"
	"github.com/timandy/GoMybatis/v7/utils"
)

type LogSystem struct {
	log     Log
	started bool
}

//logImpl:日志实现类,queueLen:消息队列缓冲长度
func (it LogSystem) New(logImpl Log) (LogSystem, error) {
	if it.started {
		return it, utils.NewError("LogSystem", "log system is started!")
	}
	if logImpl == nil {
		logImpl = &LogStandard{}
	}
	it.log = logImpl
	it.started = true
	return it, nil
}

//关闭日志系统和队列
func (it *LogSystem) Close() error {
	it.started = false
	return nil
}

//日志发送者
func (it *LogSystem) SendLog(logs ...string) error {
	if !it.started {
		return utils.NewError("LogSystem", "no log writer! you must call go GoMybatis.LogSystem{}.New()")
	}
	var buf bytes.Buffer
	for _, v := range logs {
		buf.WriteString(v)
	}
	it.log.Println(buf.Bytes())
	return nil
}
