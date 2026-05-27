package stmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildStmtConvert_MysqlFamily(t *testing.T) {
	for _, drv := range []string{"mysql", "mymysql", "mssql", "sqlite3"} {
		c, err := BuildStmtConvert(drv)
		assert.NoError(t, err)
		assert.IsType(t, &MysqlStmtIndexConvertImpl{}, c, "driver %q should map to MysqlStmtIndexConvertImpl", drv)
	}
}

func TestBuildStmtConvert_PostgresFamily(t *testing.T) {
	for _, drv := range []string{"postgres", "kingbase"} {
		c, err := BuildStmtConvert(drv)
		assert.NoError(t, err)
		assert.IsType(t, &PostgreStmtIndexConvertImpl{}, c, "driver %q should map to PostgreStmtIndexConvertImpl", drv)
	}
}

func TestBuildStmtConvert_Oracle(t *testing.T) {
	c, err := BuildStmtConvert("oci8")
	assert.NoError(t, err)
	assert.IsType(t, &OracleStmtIndexConvertImpl{}, c)
}

func TestBuildStmtConvert_UnknownDriverPanics(t *testing.T) {
	assert.PanicsWithValue(t,
		"[GoMybatis] un support dbName:nosuchdb only support: mysql,mymysql,mssql,sqlite3,postgres,kingbase,oci8",
		func() { _, _ = BuildStmtConvert("nosuchdb") })
}
