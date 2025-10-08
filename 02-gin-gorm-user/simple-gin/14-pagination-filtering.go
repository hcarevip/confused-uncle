/*
 * File: 14-pagination-filtering.go
 * Date: 2025/9/9 23:46
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// SortDir 定义的是新的命名类型
type SortDir string

const (
	SortAsc  SortDir = "asc"
	SortDesc SortDir = "desc"
)

type PageParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
	Limit    int `json:"limit"`
}

type SortParams struct {
	SortBy     string  `json:"sort_by"` // logical field name, whitelist only
	SortDir    SortDir `json:"sort_dir"`
	SortColumn string  `json:"-"`
}

type FilterParams struct {
	Status      string     `json:"status"`       // e.g. active/inactive/...
	CreatedFrom *time.Time `json:"created_from"` // ISO date
	CreatedTo   *time.Time `json:"created_to"`   // ISO date
	Keyword     string     `json:"kw"`           // free-text keyword, length capped, safe charset
}

type QueryParams struct {
	Page     PageParams   `json:"page"`
	Sort     SortParams   `json:"sort"`
	Filter   FilterParams `json:"filter"`
	RawQuery url.Values   `json:"-"` // for debugging
}

type Policy struct {
	DefaultPage       int
	DefaultPageSize   int
	MaxPageSize       int
	AllowedSortFields map[string]string // logical -> DB column (safe)
	AllowedStatus     map[string]struct{}
	MaxKeywordLen     int
	KeywordSafeRunes  func(r rune) bool // char whitelist
}

func DefaultPolicy() Policy {
	return Policy{
		DefaultPage:     1,
		DefaultPageSize: 20,
		MaxPageSize:     100,
		AllowedSortFields: map[string]string{
			"created_at": "created_at",
			"username":   "username",
			"status":     "status",
			"id":         "id",
		},
		AllowedStatus: map[string]struct{}{
			"active":   {},
			"inactive": {},
			"banned":   {},
		},
		MaxKeywordLen: 64,
		KeywordSafeRunes: func(r rune) bool {
			// 仅允许字母数字、空格、常见汉字范围、部分标点
			if r == ' ' || r == '-' || r == '_' || r == '.' {
				return true
			}
			if r >= '0' && r <= '9' {
				return true
			}
			if r >= 'a' && r <= 'z' {
				return true
			}
			if r >= 'A' && r <= 'Z' {
				return true
			}
			// 汉字常见范围
			if r >= 0x4E00 && r <= 0x9FA5 {
				return true
			}
			return false
		},
	}
}
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
func parseISODate(s string) (time.Time, error) {
	// 支持 YYYY-MM-DD 或 RFC3339
	if len(s) == 10 {
		return time.Parse("2006-01-02", s)
	}
	return time.Parse(time.RFC3339, s)
}
func (q *QueryParams) OrderClause() string {
	dir := "DESC"
	if q.Sort.SortDir == SortAsc {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", q.Sort.SortColumn, dir)
}

func ParseQuery(ctx *gin.Context, pol Policy) (QueryParams, error) {
	// map[created_from:[2025-01-01] created_to:[2025-10-01] kw:[abc] status:[active]]
	q := ctx.Request.URL.Query()
	res := QueryParams{RawQuery: q}
	page := parseInt(q.Get("page"), pol.DefaultPage)
	if page < 1 {
		page = pol.DefaultPage
	}
	pageSize := parseInt(q.Get("page_size"), pol.DefaultPageSize)
	if pageSize < 1 {
		pageSize = pol.DefaultPageSize
	}
	if pageSize > pol.MaxPageSize {
		pageSize = pol.MaxPageSize
	}
	res.Page.Page = page
	res.Page.PageSize = pageSize
	res.Page.Offset = (page - 1) * pageSize // page=3  3-1  2 x pageSize
	res.Page.Limit = pageSize
	sortBy := strings.TrimSpace(q.Get("sort_by"))
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortColumn, ok := pol.AllowedSortFields[sortBy]
	if !ok {
		return QueryParams{}, fmt.Errorf("invalid sort_by: %s", sortBy)
	}
	dir := SortDir(strings.ToLower(strings.TrimSpace(q.Get("sort_dir"))))
	if dir != SortAsc && dir != SortDesc && dir != "" {
		return QueryParams{}, fmt.Errorf("invalid sort_dir: %s", dir)
	}
	if dir == "" {
		dir = SortDesc // 默认desc
	}
	res.Sort.SortBy = sortBy
	res.Sort.SortDir = dir
	res.Sort.SortColumn = sortColumn
	status := strings.TrimSpace(q.Get("status"))
	if status != "" {
		if _, ok := pol.AllowedStatus[status]; !ok {
			return QueryParams{}, fmt.Errorf("invalid status: %s", status)
		}
	}
	var fromPtr, toPtr *time.Time
	if s := strings.TrimSpace(q.Get("created_from")); s != "" {
		t, err := parseISODate(s)
		if err != nil {
			return QueryParams{}, fmt.Errorf("invalid created_from: %v", err)
		}
		fromPtr = &t
	}
	if s := strings.TrimSpace(q.Get("created_to")); s != "" {
		t, err := parseISODate(s)
		if err != nil {
			return QueryParams{}, fmt.Errorf("invalid created_to: %v", err)
		}
		toPtr = &t
	}
	if fromPtr != nil && toPtr != nil && fromPtr.After(*toPtr) {
		return QueryParams{}, errors.New("created_from > created_to")
	}

	// keyword
	kw := strings.TrimSpace(q.Get("kw"))
	if kw != "" {
		if utf8.RuneCountInString(kw) > pol.MaxKeywordLen {
			return QueryParams{}, fmt.Errorf("kw too long: max %d", pol.MaxKeywordLen)
		}
		for _, r := range kw {
			if !pol.KeywordSafeRunes(r) {
				return QueryParams{}, fmt.Errorf("kw has unsafe char: %q", r)
			}
		}
	}
	res.Filter = FilterParams{
		Status:      status,
		CreatedFrom: fromPtr,
		CreatedTo:   toPtr,
		Keyword:     kw,
	}
	return res, nil
}

func GetListUsers(ctx *gin.Context) {
	pol := DefaultPolicy()
	params, err := ParseQuery(ctx, pol)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"code": "INVALID_QUERY", "msg": err.Error(),
		})
		return
	}
	//  DB 伪代码：所有条件走参数化
	ctx.JSON(http.StatusOK, gin.H{
		"page":   params.Page,
		"sort":   params.Sort,
		"filter": params.Filter,
	})
}

func RegisterPaginationRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/test")
	{
		// ?status=active&created_from=2025-01-01&created_to=2025-10-01&kw=abc
		api.GET("/user/list", GetListUsers)
	}
}
