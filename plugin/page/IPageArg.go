package page

// IPageArg 分页参数接口
type IPageArg interface {
	GetPageNum() int  //必须大于等于1
	GetPageSize() int //必须大于等于1
}

// Assert 校验分页参数
func Assert(pa IPageArg) {
	if pa.GetPageNum() <= 0 {
		panic("GetPageNum() out of range")
	}
	if pa.GetPageSize() <= 0 {
		panic("GetPageSize() out of range")
	}
}

// GetOffset 计算 offset
func GetOffset(pa IPageArg) int {
	return (pa.GetPageNum() - 1) * pa.GetPageSize()
}

// GetLimit 计算 limit
func GetLimit(pa IPageArg) int {
	return pa.GetPageSize()
}
