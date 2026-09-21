package pagination

import (
	"net/url"
	"testing"
)

func TestParseAndSlice(t *testing.T) {
	p, err := Parse(url.Values{"page": {"2"}, "page_size": {"2"}, "order": {"desc"}})
	if err != nil || !p.Desc || p.Page != 2 || p.PageSize != 2 {
		t.Fatalf("params=%+v err=%v", p, err)
	}
	got := Slice([]string{"a", "b", "c"}, p)
	if len(got.Items) != 1 || got.Items[0] != "c" || got.Total != 3 {
		t.Fatalf("result=%+v", got)
	}
}

func TestParseRejectsInvalidValues(t *testing.T) {
	if _, err := Parse(url.Values{"page_size": {"101"}}); !Invalid(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestSliceHandlesHugePageWithoutOverflow(t *testing.T) {
	got := Slice([]string{"a"}, Params{Page: int(^uint(0) >> 1), PageSize: 20})
	if len(got.Items) != 0 || got.Total != 1 {
		t.Fatalf("result=%+v", got)
	}
}
