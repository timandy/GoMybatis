package GoMybatis

import (
	"database/sql"

	"github.com/timandy/GoMybatis/v7/ast"
	"github.com/timandy/GoMybatis/v7/stmt"
	"github.com/timandy/GoMybatis/v7/tx"
)

type Result struct {
	LastInsertId int64
	RowsAffected int64
}

type Session interface {
	Id() string
	Query(sqlorArgs string) ([]map[string][]byte, error)
	Exec(sqlorArgs string) (*Result, error)
	//Prepare sql, example sqlPrepare: select * from table where id = ?   ,   args：'1'
	QueryPrepare(sqlPrepare string, args ...interface{}) ([]map[string][]byte, error)
	//Prepare sql, example sqlPrepare: select * from table where id = ?   ,   args：'1'
	ExecPrepare(sqlPrepare string, args ...interface{}) (*Result, error)
	Rollback() error
	Commit() error
	Begin(p *tx.Propagation) error
	Close()
	LastPROPAGATION() *tx.Propagation
	StmtConvert() (stmt.StmtIndexConvert, error)
}

//产生session的引擎
type SessionEngine interface {
	//打开数据库
	Open(driverName, dataSourceLink string) (*sql.DB, error)
	//写方法到mapper
	WriteMapperPtr(ptr interface{}, xml []byte)
	//引擎名称
	Name() string
	//创建session
	NewSession(mapperName string) (Session, error)
	//获取数据源路由
	DataSourceRouter() DataSourceRouter
	//设置数据源路由
	SetDataSourceRouter(router DataSourceRouter)

	//是否启用日志
	LogEnable() bool

	//是否启用日志
	SetLogEnable(enable bool)

	//获取日志实现类
	Log() Log

	//设置日志实现类
	SetLog(log Log)

	//session工厂
	SessionFactory() *SessionFactory

	//设置session工厂
	SetSessionFactory(factory *SessionFactory)

	//设置sql类型转换器
	SetSqlArgTypeConvert(convert ast.SqlArgTypeConvert)

	//表达式执行引擎
	ExpressionEngine() ast.ExpressionEngine

	//设置表达式执行引擎
	SetExpressionEngine(engine ast.ExpressionEngine)

	//sql构建器
	SqlBuilder() SqlBuilder

	//设置sql构建器
	SetSqlBuilder(builder SqlBuilder)

	//sql查询结果解析器
	SqlResultDecoder() SqlResultDecoder

	//设置sql查询结果解析器
	SetSqlResultDecoder(decoder SqlResultDecoder)

	//模板解析器
	TemplateDecoder() TemplateDecoder

	//设置模板解析器
	SetTemplateDecoder(decoder TemplateDecoder)

	//（注意（该方法需要在多协程环境下调用）启用会从栈获取协程id，有一定性能消耗，换取最大的事务定义便捷）
	GoroutineSessionMap() *GoroutineSessionMap

	//是否启用goroutineIDEnable（注意（该方法需要在多协程环境下调用）启用会从栈获取协程id，有一定性能消耗，换取最大的事务定义便捷）
	SetGoroutineIDEnable(enable bool)

	//是否启用goroutineIDEnable（注意（该方法需要在多协程环境下调用）启用会从栈获取协程id，有一定性能消耗，换取最大的事务定义便捷）
	GoroutineIDEnable() bool

	IsPrintWarning() bool

	// SetPanicOnError 设置执行SQL失败时是否panic
	SetPanicOnError(value bool)

	// IsPanicOnError 获取执行SQL失败时是否panic
	IsPanicOnError() bool

	// SetWriteBackAutoField 设置 insert() 时是否写回自增列
	SetWriteBackAutoField(value bool)

	// IsWriteBackAutoFiled 获取 insert() 时是否写回自增列
	IsWriteBackAutoFiled() bool

	// GetProperty 获取属性
	GetProperty(key string) any

	// SetProperty 设置属性
	SetProperty(key string, value any)

	// RemoveProperty 删除属性
	RemoveProperty(key string)
}
