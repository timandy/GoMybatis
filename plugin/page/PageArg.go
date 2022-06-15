package page

// PageArg 分页参数实现. 业务层可自定义, 需要实现 IPageArg 接口
//goland:noinspection GoNameStartsWithPackageName
type PageArg struct {
	PageNum  int `json:"page_num"`
	PageSize int `json:"page_size"`
}

func (p PageArg) GetPageNum() int {
	return p.PageNum
}

func (p PageArg) GetPageSize() int {
	return p.PageSize
}
