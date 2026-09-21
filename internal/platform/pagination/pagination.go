package pagination

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Params describes the common list query controls. Page numbers are one-based.
type Params struct {
	Page     int
	PageSize int
	Query    string
	Sort     string
	Desc     bool
}

func Parse(values url.Values) (Params, error) {
	p := Params{Page: 1, PageSize: DefaultPageSize, Query: strings.TrimSpace(values.Get("q")), Sort: strings.TrimSpace(values.Get("sort"))}
	if raw := values.Get("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return Params{}, errInvalid
		}
		p.Page = v
	}
	if raw := values.Get("page_size"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxPageSize {
			return Params{}, errInvalid
		}
		p.PageSize = v
	}
	if raw := values.Get("order"); raw != "" {
		switch strings.ToLower(raw) {
		case "asc":
		case "desc":
			p.Desc = true
		default:
			return Params{}, errInvalid
		}
	}
	return p, nil
}

var errInvalid = &invalidError{}

type invalidError struct{}

func (*invalidError) Error() string { return "invalid pagination parameters" }

func Invalid(err error) bool { return err == errInvalid }

type Result[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

func Slice[T any](items []T, p Params) Result[T] {
	if items == nil {
		items = []T{}
	}
	if p.Page < 1 || p.PageSize < 1 {
		return Result[T]{Items: []T{}, Page: p.Page, PageSize: p.PageSize, Total: len(items)}
	}
	// Check before multiplying so a client-supplied page cannot overflow int.
	if p.Page-1 > len(items)/p.PageSize {
		return Result[T]{Items: []T{}, Page: p.Page, PageSize: p.PageSize, Total: len(items)}
	}
	start := (p.Page - 1) * p.PageSize
	if start >= len(items) {
		return Result[T]{Items: []T{}, Page: p.Page, PageSize: p.PageSize, Total: len(items)}
	}
	end := start + p.PageSize
	if end > len(items) {
		end = len(items)
	}
	return Result[T]{Items: items[start:end], Page: p.Page, PageSize: p.PageSize, Total: len(items)}
}
