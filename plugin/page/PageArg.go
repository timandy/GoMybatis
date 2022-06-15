package page

// PageArg 分页参数实现. 业务层可自定义, 需要实现 IPageArg 接口
//goland:noinspection GoNameStartsWithPackageName
type PageArg struct {
	PageNum  int `json:"page_num"`
	PageSize int `json:"page_size"`
}

// GetPageNum 获取第几页, 从1开始, 接收器不能使用指针
func (p PageArg) GetPageNum() int {
	return p.PageNum
}

// GetPageSize 获取页大小, 从1开始, 接收器不能使用指针
func (p PageArg) GetPageSize() int {
	return p.PageSize
}
