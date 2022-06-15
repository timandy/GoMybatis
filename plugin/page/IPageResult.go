package page

// IPageResult 分页结果
type IPageResult interface {
	SetTotalCount(totalCount int64)   //不分页情况下的总行数
	SetPageCount(pageCount int)       //总页数
	SetDisplayCount(displayCount int) //当前页行数
	//泛型接口不方便判断, 所以放到实现中
	//SetList(list []T)
}
