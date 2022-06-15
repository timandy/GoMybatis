package page

// PageResult 分页结果实现. 业务层可自定义, 需要实现 IPageResult 接口并添加 SetList([]T) 方法
//goland:noinspection GoNameStartsWithPackageName
type PageResult[T any] struct {
	TotalCount   int64 `json:"total_count"`
	PageCount    int   `json:"page_count"`
	DisplayCount int   `json:"display_count"`
	List         []T   `json:"list"`
}

func (p *PageResult[T]) SetTotalCount(totalCount int64) {
	p.TotalCount = totalCount
}

func (p *PageResult[T]) SetPageCount(pageCount int) {
	p.PageCount = pageCount
}

func (p *PageResult[T]) SetDisplayCount(displayCount int) {
	p.DisplayCount = displayCount
}

// SetList 自定义实现时必须有该方法
func (p *PageResult[T]) SetList(list []T) {
	p.List = list
}
