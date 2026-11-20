package handler

type PageParams struct {
	Page     int `query:"page" minimum:"1" default:"1" doc:"1-based page number"`
	PageSize int `query:"pageSize" minimum:"1" maximum:"100" default:"25" doc:"Items per page"`
}

func (p PageParams) Limit() int {
	return p.PageSize
}

func (p PageParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type Page[T any] struct {
	Items    []T  `json:"items" nullable:"false"`
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
	Total    int  `json:"total"`
	HasNext  bool `json:"hasNext"`
}

type PageOutput[T any] struct {
	Body Page[T]
}

func NewPage[T any](items []T, params PageParams, total int) Page[T] {
	if items == nil {
		items = []T{}
	}

	return Page[T]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		HasNext:  params.Offset()+len(items) < total,
	}
}

func mapItems[From, To any](items []From, mapItem func(From) To) []To {
	mapped := make([]To, 0, len(items))

	for _, item := range items {
		mapped = append(mapped, mapItem(item))
	}

	return mapped
}
