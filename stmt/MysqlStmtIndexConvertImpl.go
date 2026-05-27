package stmt

var _ StmtIndexConvert = (*MysqlStmtIndexConvertImpl)(nil)

type MysqlStmtIndexConvertImpl struct {
}

func (it *MysqlStmtIndexConvertImpl) Convert() string {
	return " ? "
}

func (it *MysqlStmtIndexConvertImpl) Inc() {

}

func (it *MysqlStmtIndexConvertImpl) Get() int {
	return 0
}
