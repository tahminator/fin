package handler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tahminator/fin/internal/handler"
)

func TestPageParamsOffsetAndLimit(t *testing.T) {
	tests := []struct {
		name   string
		params handler.PageParams
		offset int
	}{
		{name: "first page starts at zero", params: handler.PageParams{Page: 1, PageSize: 25}, offset: 0},
		{name: "second page skips one page", params: handler.PageParams{Page: 2, PageSize: 25}, offset: 25},
		{name: "fourth page of ten skips thirty", params: handler.PageParams{Page: 4, PageSize: 10}, offset: 30},
		{name: "page size of one skips page minus one", params: handler.PageParams{Page: 7, PageSize: 1}, offset: 6},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.params.Offset(); got != tc.offset {
				t.Errorf("Offset() = %d, want %d", got, tc.offset)
			}

			if got := tc.params.Limit(); got != tc.params.PageSize {
				t.Errorf("Limit() = %d, want %d", got, tc.params.PageSize)
			}
		})
	}
}

func TestNewPageHasNext(t *testing.T) {
	tests := []struct {
		name    string
		items   int
		params  handler.PageParams
		total   int
		hasNext bool
	}{
		{name: "full first page with more remaining", items: 10, params: handler.PageParams{Page: 1, PageSize: 10}, total: 25, hasNext: true},
		{name: "partial last page", items: 5, params: handler.PageParams{Page: 3, PageSize: 10}, total: 25, hasNext: false},
		{name: "full last page exactly reaching total", items: 10, params: handler.PageParams{Page: 2, PageSize: 10}, total: 20, hasNext: false},
		{name: "one item beyond full page", items: 10, params: handler.PageParams{Page: 2, PageSize: 10}, total: 21, hasNext: true},
		{name: "empty page past the end", items: 0, params: handler.PageParams{Page: 5, PageSize: 10}, total: 20, hasNext: false},
		{name: "empty result", items: 0, params: handler.PageParams{Page: 1, PageSize: 25}, total: 0, hasNext: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page := handler.NewPage(make([]int, tc.items), tc.params, tc.total)

			if page.HasNext != tc.hasNext {
				t.Errorf("HasNext = %v, want %v", page.HasNext, tc.hasNext)
			}

			if page.Page != tc.params.Page || page.PageSize != tc.params.PageSize || page.Total != tc.total {
				t.Errorf("page = %+v, want page %d size %d total %d", page, tc.params.Page, tc.params.PageSize, tc.total)
			}

			if len(page.Items) != tc.items {
				t.Errorf("len(Items) = %d, want %d", len(page.Items), tc.items)
			}
		})
	}
}

func TestNewPageNilItemsBecomeEmptySlice(t *testing.T) {
	page := handler.NewPage[string](nil, handler.PageParams{Page: 1, PageSize: 25}, 0)

	if page.Items == nil {
		t.Fatal("Items is nil, want non-nil empty slice")
	}

	body, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	if !strings.Contains(string(body), `"items":[]`) {
		t.Errorf("body = %s, want items encoded as []", body)
	}
}
