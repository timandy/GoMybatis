package example

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/timandy/GoMybatis/v7"
	"github.com/timandy/GoMybatis/v7/ids"
)

//支持基本类型和指针(int,string,time.Time,float...且需要指定参数名称`args:"name"以逗号隔开，且位置要和实际参数相同)
//参数中包含有*GoMybatis.Session的类型，用于自定义事务，也可以选择例如TestService 这样使用更简单的声明式事务
//自定义结构体参数（属性必须大写）
//方法 return 必须包含有error ,为了返回错误信息
type ExampleActivityMapper struct {
	//声明书事务 查看 struct TestService 中的定义
	GoMybatis.SessionSupport                                   //session事务操作 写法1.  ExampleActivityMapper.SessionSupport.NewSession()
	NewSession               func() (GoMybatis.Session, error) //session事务操作.写法2   ExampleActivityMapper.NewSession()

	//模板示例
	SelectTemplate      func(name string) ([]Activity, error) `args:"name"`
	SelectCountTemplate func(name string) (int64, error)      `args:"name"`
	InsertTemplate      func(arg Activity) (int64, error)
	InsertTemplateBatch func(args []Activity) (int64, error) `args:"args"`
	UpdateTemplate      func(arg Activity) (int64, error)    `args:"name"`
	DeleteTemplate      func(name string) (int64, error)     `args:"name"`

	//传统mybatis示例
	SelectByIds       func(ids []string) ([]Activity, error)       `args:"ids"`
	SelectByIdMaps    func(ids map[int]string) ([]Activity, error) `args:"ids"`
	SelectAll         func() ([]map[string]string, error)
	SelectByCondition func(name string, startTime *time.Time, endTime *time.Time, page *int, size *int) ([]Activity, error) `args:"name,startTime,endTime,page,size"`
	UpdateById        func(session *GoMybatis.Session, arg Activity) (int64, error)
	Insert            func(arg Activity) (int64, error)
	CountByCondition  func(name string, startTime time.Time, endTime time.Time) (int, error) `args:"name,startTime,endTime"`
	DeleteById        func(id string) (int64, error)                                         `args:"id"`
	Choose            func(deleteFlag int) ([]Activity, error)                               `args:"deleteFlag"`
	SelectLinks       func(column string) ([]Activity, error)                                `args:"column"`
}

var engine *GoMybatis.GoMybatisEngine

//初始化mapper文件和结构体
var exampleActivityMapper = ExampleActivityMapper{}

//测试服务下，声明式事务
type TestService struct {
	exampleActivityMapper *ExampleActivityMapper //服务包含一个mapper操作数据库，类似java spring mvc

	//类似拷贝spring MVC的声明式事务注解
	//rollback:回滚操作为error类型(你也可以自定义实现了builtin.error接口的自定义struct，框架会把自定义的error类型转换为string，检查是否包含，是则回滚
	//tx:"" 开启事务，`tx:"PROPAGATION_REQUIRED,error"` 指定传播行为为REQUIRED(默认REQUIRED))
	UpdateName   func(id string, name string) error   `tx:"" rollback:"error"`
	UpdateRemark func(id string, remark string) error `tx:"" rollback:"error"`
}

//推荐使用snowflake雪花算法 代替uuid防止ID碰撞
var SnowflakeNode, e = ids.NewNode(0)

func init() {
	if MysqlUri == "*" {
		println("GoMybatisEngine not init! because MysqlUri is * or MysqlUri is ''")
		return
	}
	engine = GoMybatis.NewEngine()

	//设置打印自动生成的xml 到控制台方便调试，false禁用
	engine.TemplateDecoder().SetPrintElement(false)
	//设置是否打印警告(建议开启)
	engine.SetPrintWarning(false)

	//mysql链接格式为         用户名:密码@(数据库链接地址:端口)/数据库名称   例如root:123456@(***.mysql.rds.aliyuncs.com:3306)/test
	_, err := engine.Open("mysql", MysqlUri) //此处请按格式填写你的mysql链接，这里用*号代替
	if err != nil {
		panic(err.Error())
	}

	//动态数据源路由(可选)
	/**
	engine.Open("mysql", MysqlUri)//添加第二个mysql数据库,请把MysqlUri改成你的第二个数据源链接
	var router = GoMybatis.GoMybatisDataSourceRouter{}.New(func(mapperName string) *string {
		//根据包名路由指向数据源
		if strings.Contains(mapperName, "example.") {
			var url = MysqlUri//第二个mysql数据库,请把MysqlUri改成你的第二个数据源链接
			fmt.Println(url)
			return &url
		}
		return nil
	})
	engine.SetDataSourceRouter(&router)
	**/

	//自定义日志实现(可选)
	//engine.SetLogEnable(true)
	//engine.SetLog(&GoMybatis.LogStandard{
	//	PrintlnFunc: func(messages ...string) {
	//		println("log>> ", fmt.Sprint(messages))
	//	},
	//})
	//读取mapper xml文件
	bytes, _ := ioutil.ReadFile("Example_ActivityMapper.xml")
	//设置对应的mapper xml文件
	engine.WriteMapperPtr(&exampleActivityMapper, bytes)
}

//插入
func Test_inset(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//推荐使用snowflake雪花算法 代替uuid防止ID碰撞,id最好用string类型. 否则客户端有可能不支持long类型例如JavaScript
	var id = SnowflakeNode.Generate().String()
	//使用mapper
	var result, err = exampleActivityMapper.Insert(Activity{Id: id, Name: "test_insert", PcLink: "ssss", CreateTime: time.Now(), DeleteFlag: 1})
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//修改
func Test_update(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	var activityBean = Activity{
		Id:   "171",
		Name: "rs168",
	}
	var updateNum, e = exampleActivityMapper.UpdateById(nil, activityBean) //sessionId 有值则使用已经创建的session，否则新建一个session
	fmt.Println("updateNum=", updateNum)
	if e != nil {
		panic(e)
	}
}

//删除
func Test_delete(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.DeleteById("171")
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//查询
func Test_select(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	name := ""

	var result, err = exampleActivityMapper.SelectByCondition(name, nil, nil, nil, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//查询
func Test_select_all(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.SelectAll()
	if err != nil {
		panic(err)
	}
	var b, _ = json.Marshal(result)
	fmt.Println("result=", string(b))
}

//查询
func Test_count(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.CountByCondition("", time.Time{}, time.Time{})
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//本地GoMybatis使用例子
func Test_ForEach(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var ids = []string{"1", "2"}
	var result, err = exampleActivityMapper.SelectByIds(ids)
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//本地GoMybatis使用例子
func Test_ForEach_Map(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var ids = map[int]string{1: "165", 2: "166"}
	var result, err = exampleActivityMapper.SelectByIdMaps(ids)
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//本地事务使用例子
func Test_local_Transation(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用事务
	var session, err = exampleActivityMapper.SessionSupport.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	session.Begin(nil) //开启事务
	var activityBean = Activity{
		Id:         "170",
		Name:       "rs168-8",
		DeleteFlag: 1,
	}
	var updateNum, e = exampleActivityMapper.UpdateById(&session, activityBean) //sessionId 有值则使用已经创建的session，否则新建一个session
	fmt.Println("updateNum=", updateNum)
	if e != nil {
		panic(e)
	}
	session.Commit() //提交事务
	session.Close()  //关闭事务
}

func Test_choose(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.Choose(1)
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//查询
func Test_include_sql(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.SelectLinks("name")
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

func TestSelectTemplate(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.SelectTemplate("hello")
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

func TestSelectCountTemplate(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.SelectCountTemplate("hello")
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

func TestInsertTemplate(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//使用mapper
	var result, err = exampleActivityMapper.InsertTemplate(Activity{Id: "178", Name: "test_insret", CreateTime: time.Now(), Sort: 1, Status: 1, DeleteFlag: 1})
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//批量插入模板
func TestInsertTemplateBatch(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	var args = []Activity{
		{
			Id:         "221",
			Name:       "test",
			CreateTime: time.Now(),
		},
		{
			Id:         "222",
			Name:       "test",
			CreateTime: time.Now(),
		},
		{
			Id:         "223",
			Name:       "test",
			CreateTime: time.Now(),
		},
	}
	n, err := exampleActivityMapper.InsertTemplateBatch(args)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("updateNum", n)
	time.Sleep(time.Second)
}

//修改模板默认支持逻辑删除和乐观锁
func TestUpdateTemplate(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	var activityBean = Activity{
		Id:      "171",
		Name:    "rs168",
		Version: 2,
	}
	//会自动生成乐观锁和逻辑删除字段 set version= * where version = * and delete_flag = *
	// update set name = 'rs168',version = 1 from biz_activity where name = 'rs168' and delete_flag = 1 and version = 0
	var updateNum, e = exampleActivityMapper.UpdateTemplate(activityBean)
	fmt.Println("updateNum=", updateNum)
	if e != nil {
		panic(e)
	}
}

//删除
func TestDeleteTemplate(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	//模板默认支持逻辑删除
	var result, err = exampleActivityMapper.DeleteTemplate("rs168")
	if err != nil {
		panic(err)
	}
	fmt.Println("result=", result)
}

//嵌套事务/带有传播行为的事务
func TestTestService(t *testing.T) {
	if MysqlUri == "" || MysqlUri == "*" {
		fmt.Println("no database url define in Example_config.go , you must set the mysql link!")
		return
	}
	var testService = initTestService()

	//go testService.UpdateName("167", "updated name1")
	testService.UpdateName("167", "updated name2")

	time.Sleep(3 * time.Second)
}

//嵌套事务服务TestService
func initTestService() TestService {
	var testService TestService
	testService = TestService{
		exampleActivityMapper: &exampleActivityMapper,
		UpdateRemark: func(id string, remark string) error {
			var activitys, err = testService.exampleActivityMapper.SelectByIds([]string{id})
			if err != nil {
				panic(err)
			}
			//TODO 此处可能会因为activitys长度为0 导致数组越界 painc,painc 为运行时异常 框架自动回滚事务
			var activity = activitys[0]
			activity.Remark = remark
			updateNum, err := testService.exampleActivityMapper.UpdateTemplate(activity)
			if err != nil {
				panic(err)
			}
			println("UpdateRemark:", updateNum)
			if id == "167" {
				return errors.New("e")
			}
			return nil
		},
		UpdateName: func(id string, name string) error {
			var activitys, err = testService.exampleActivityMapper.SelectByIds([]string{id})
			if err != nil {
				panic(err)
			}
			var activity = activitys[0]
			activity.Name = name
			updateNum, err := testService.exampleActivityMapper.UpdateTemplate(activity)
			if err != nil {
				panic(err)
			}
			println("UpdateName:", updateNum)
			testService.UpdateRemark("172", "p2")
			testService.UpdateRemark("167", "p1")
			return nil
		},
	}
	GoMybatis.AopProxyService(&testService, engine)
	return testService
}

func TestCreateDefaultXmlWriteToFile(t *testing.T) {
	var bean = Activity{} //此处只是举例，应该替换为你自己的数据库模型
	GoMybatis.OutPutXml(reflect.TypeOf(bean).Name()+"Mapper.xml", GoMybatis.CreateXml("biz_"+GoMybatis.StructToSnakeString(bean), bean))
}
