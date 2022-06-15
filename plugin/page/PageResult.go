package page

// PageResult 分页结果实现. 业务层可自定义, 需要实现 IPageResult 接口并添加 SetList([]T) 方法
//goland:noinspection GoNameStartsWithPackageName
type PageResult[T any] struct {
	TotalCount   int64 `json:"total_count"`
	PageCount    int   `json:"page_count"`
	DisplayCount int   `json:"display_count"`
	List         []T   `json:"list"`
}

// SetTotalCount 设置总行数, 接收器必须使用指针
func (p *PageResult[T]) SetTotalCount(totalCount int64) {
	p.TotalCount = totalCount
}

// SetPageCount 设置总页数, 接收器必须使用指针
func (p *PageResult[T]) SetPageCount(pageCount int) {
	p.PageCount = pageCount
}

// SetDisplayCount 设置当前页行数, 接收器必须使用指针
func (p *PageResult[T]) SetDisplayCount(displayCount int) {
	p.DisplayCount = displayCount
}

// SetList 设置当前页数据, 接收器必须使用指针 [!!!自定义实现时必须有该方法!!!]
func (p *PageResult[T]) SetList(list []T) {
	p.List = list
}
