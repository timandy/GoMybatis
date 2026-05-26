package GoMybatis

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/timandy/GoMybatis/v7/engines"
	"github.com/timandy/GoMybatis/v7/lib/github.com/beevik/etree"
	"github.com/timandy/GoMybatis/v7/stmt"
	"github.com/timandy/GoMybatis/v7/utils"
)

//压力测试 sql构建情况
func Benchmark_SqlBuilder(b *testing.B) {
	b.StopTimer()
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <!--List<Activity> selectByCondition(@Param("name") String name,@Param("startTime") Date startTime,@Param("endTime") Date endTime,@Param("index") Integer index,@Param("size") Integer size);-->
    <!-- 后台查询产品 -->
    <select id="selectByCondition">
        select * from biz_activity where delete_flag=1
        <if test="name != nil">
            and name like concat('%',#{name},'%')
        </if>
        <if test="startTime != nil">
            and create_time >= #{startTime}
        </if>
        <if test="endTime != nil">
            and create_time &lt;= #{endTime}
        </if>
        order by create_time desc
        <if test="page >= 0 and size != 0">limit #{page}, #{size}</if>
    </select>
</mapper>`

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, false)

	var mapperTree = LoadMapperXml([]byte(mapper))
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["name"] = ""
	paramMap["startTime"] = ""
	paramMap["endTime"] = ""
	paramMap["page"] = 0
	paramMap["size"] = 0

	//paramMap["func_name != nil"] = func(arg map[string]interface{}) interface{} {
	//	return arg["name"] != nil
	//}
	//paramMap["func_startTime != nil"] = func(arg map[string]interface{}) interface{} {
	//	return arg["startTime"] != nil
	//}
	//paramMap["func_endTime != nil"] = func(arg map[string]interface{}) interface{} {
	//	return arg["endTime"] != nil
	//}
	//paramMap["func_page >= 0 and size != 0"] = func(arg map[string]interface{}) interface{} {
	//	return arg["page"] != nil && arg["size"] != nil
	//}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var array = []interface{}{}
		_, e := builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
		if e != nil {
			b.Fatal(e)
		}
	}
}

//测试sql生成tps
func Test_SqlBuilder_Tps(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <!--List<Activity> selectByCondition(@Param("name") String name,@Param("startTime") Date startTime,@Param("endTime") Date endTime,@Param("index") Integer index,@Param("size") Integer size);-->
    <!-- 后台查询产品 -->
    <select id="selectByCondition">
        select * from biz_activity where delete_flag=1
        <if test="name != nil">
            and name like concat('%',#{name},'%')
        </if>
        <if test="startTime != nil">
            and create_time >= #{startTime}
        </if>
        <if test="endTime != nil">
            and create_time &lt;= #{endTime}
        </if>
        order by create_time desc
        <if test="page >= 0 and size != 0">limit #{page}, #{size}</if>
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, false)
	var paramMap = make(map[string]interface{})
	paramMap["name"] = ""
	paramMap["startTime"] = ""
	paramMap["endTime"] = ""
	paramMap["page"] = 0
	paramMap["size"] = 0

	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var startTime = time.Now()
	for i := 0; i < 100000; i++ {
		//var sql, e =
		var array = []interface{}{}
		_, e := builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
		if e != nil {
			t.Fatal(e)
		}
		//fmt.Println(sql, e)
	}
	utils.CountMethodTps(100000, startTime, "Test_SqlBuilder_Tps")
}

func TestGoMybatisSqlBuilder_BuildSql(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <resultMap id="BaseResultMap">
        <id column="id" property="id"/>
        <result column="name" property="name" langType="string"/>
        <result column="pc_link" property="pcLink" langType="string"/>
        <result column="h5_link" property="h5Link" langType="string"/>
        <result column="remark" property="remark" langType="string"/>
        <result column="create_time" property="createTime" langType="time.Time"/>
        <result column="delete_flag" property="deleteFlag" langType="int"/>
    </resultMap>
    <select id="selectByCondition" resultMap="BaseResultMap">
        <bind name="pattern" value="'%' + name + '%'"/>
        select * from biz_activity
        <where>
            <if test="name != nil">
                and name like #{pattern}
            </if>
            <if test="startTime != nil">and create_time >= #{startTime}</if>
            <if test="endTime != nil">and create_time &lt;= #{endTime}</if>
        </where>
        order by 
        <trim prefix="" suffix="" suffixOverrides=",">
            <if test="name != nil">name,</if>
        </trim>
        desc
        <choose>
            <when test="page < 1">limit 3</when>
            <when test="page > 1">limit 2</when>
            <otherwise>limit 1</otherwise>
        </choose>
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, true)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["name"] = "name"
	paramMap["type_name"] = StringType
	paramMap["startTime"] = nil
	paramMap["endTime"] = nil
	paramMap["page"] = 0
	paramMap["size"] = 0

	var array = []interface{}{}

	var sql, err = builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(sql)
}

// 复现 / 回归 bug：同一个参数 #{userId} 在 UNION ALL 中出现多次时，
// PostgreSQL/Oracle 等编号占位符方言下，旧实现会生成单一占位符却 append 多次参数。
// Token 流重构后，每个 #{x} 出现都是一个独立的 ExprToken，渲染时各自 Inc + append，
// 占位符与参数自然 1:1。
func TestGoMybatisSqlBuilder_BuildSql_DuplicateParam_Mysql(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <select id="selectByCondition">
        select * from t1 where user_id = #{userId}
        union all
        select * from t2 where user_id = #{userId}
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, true)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["userId"] = 1001

	var array = []interface{}{}
	var sql, err = builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
	assert.NoError(t, err)
	fmt.Println("sql   :", sql)
	fmt.Println("params:", array)

	assert.Equal(t, "select * from t1 where user_id = ? union all select * from t2 where user_id = ?", sql)
	assert.Equal(t, []interface{}{1001, 1001}, array)
}

func TestGoMybatisSqlBuilder_BuildSql_DuplicateParam_Postgre(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <select id="selectByCondition">
        select * from t1 where user_id = #{userId}
        union all
        select * from t2 where user_id = #{userId}
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, true)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["userId"] = 1001

	var array = []interface{}{}
	var sql, err = builder.BuildSql(paramMap, nodes, &array, &stmt.PostgreStmtIndexConvertImpl{})
	assert.NoError(t, err)
	fmt.Println("sql   :", sql)
	fmt.Println("params:", array)

	assert.Equal(t, "select * from t1 where user_id = $1 union all select * from t2 where user_id = $2", sql)
	assert.Equal(t, []interface{}{1001, 1001}, array)
}

func TestGoMybatisSqlBuilder_BuildSql_DuplicateParam_Oracle(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <select id="selectByCondition">
        select * from t1 where user_id = #{userId}
        union all
        select * from t2 where user_id = #{userId}
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, true)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["userId"] = 1001

	var array = []interface{}{}
	var sql, err = builder.BuildSql(paramMap, nodes, &array, &stmt.OracleStmtIndexConvertImpl{})
	assert.NoError(t, err)
	fmt.Println("sql   :", sql)
	fmt.Println("params:", array)

	assert.Equal(t, "select * from t1 where user_id = :val1 union all select * from t2 where user_id = :val2", sql)
	assert.Equal(t, []interface{}{1001, 1001}, array)
}

// 回归: slice 类型参数 #{ids} 在 UNION ALL 中复用时, PG 下两个 IN 子句应分别占
// 各自的占位符编号区段 (例如 IN ($1,$2,$3) ... IN ($4,$5,$6)),
// 且 arg_array 长度 = 占位符总数。
func TestGoMybatisSqlBuilder_BuildSql_DuplicateSliceParam_Postgre(t *testing.T) {
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <select id="selectByCondition">
        select * from t1 where id in #{ids}
        union all
        select * from t2 where id in #{ids}
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, true)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["ids"] = []int{10, 20, 30}

	var array = []interface{}{}
	var sql, err = builder.BuildSql(paramMap, nodes, &array, &stmt.PostgreStmtIndexConvertImpl{})
	assert.NoError(t, err)
	fmt.Println("sql   :", sql)
	fmt.Println("params:", array)

	assert.Equal(t, "select * from t1 where id in ( $1 , $2 , $3 ) union all select * from t2 where id in ( $4 , $5 , $6 )", sql)
	assert.Equal(t, []interface{}{10, 20, 30, 10, 20, 30}, array)
}

//压力测试 sql构建情况
func Benchmark_SqlBuilder_If_Element(b *testing.B) {
	b.StopTimer()
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <!--List<Activity> selectByCondition(@Param("name") String name,@Param("startTime") Date startTime,@Param("endTime") Date endTime,@Param("index") Integer index,@Param("size") Integer size);-->
    <!-- 后台查询产品 -->
    <select id="selectByCondition">
        select * from biz_activity where delete_flag=1
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
        <if test="name != nil">
        </if>
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, false)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["name"] = ""
	paramMap["startTime"] = ""
	paramMap["endTime"] = ""
	paramMap["page"] = 0
	paramMap["size"] = 0

	//paramMap["type_name"] = StringType
	//paramMap["type_startTime"] = StringType
	//paramMap["type_endTime"] = StringType
	//paramMap["type_page"] = IntType
	//paramMap["type_size"] = IntType

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var array = []interface{}{}
		builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
	}
}

//压力测试 element嵌套构建情况
func Benchmark_SqlBuilder_Nested(b *testing.B) {
	b.StopTimer()
	var mapper = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper>
    <!--List<Activity> selectByCondition(@Param("name") String name,@Param("startTime") Date startTime,@Param("endTime") Date endTime,@Param("index") Integer index,@Param("size") Integer size);-->
    <!-- 后台查询产品 -->
    <select id="selectByCondition">
        select * from biz_activity where delete_flag=1
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>
        <set>

        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
        </set>
    </select>
</mapper>`
	var mapperTree = LoadMapperXml([]byte(mapper))

	var builder = GoMybatisSqlBuilder{}.New(ExpressionEngineProxy{}.New(&engines.ExpressionEngineGoExpress{}, true), &LogStandard{}, false)
	var nodes = builder.nodeParser.Parser(mapperTree["selectByCondition"].(*etree.Element).Child)

	var paramMap = make(map[string]interface{})
	paramMap["name"] = ""
	paramMap["startTime"] = ""
	paramMap["endTime"] = ""
	paramMap["page"] = 0
	paramMap["size"] = 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var array = []interface{}{}
		_, e := builder.BuildSql(paramMap, nodes, &array, &stmt.MysqlStmtIndexConvertImpl{})
		if e != nil {
			b.Fatal(e)
		}
	}
}
