package xianyu

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "xianyu" {
		t.Errorf("Scheme = %q, want xianyu", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "xianyu" {
		t.Errorf("Identity.Binary = %q, want xianyu", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		id  string
	}{
		{"123456", "listing", "123456"},
		{"item999", "listing", "item999"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("listing", "123456")
	want := "https://www.goofish.com/item?id=123456"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocate_UnknownType(t *testing.T) {
	_, err := Domain{}.Locate("page", "abc")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

func TestValidSort(t *testing.T) {
	for _, s := range []string{"new", "price-asc", "price-desc"} {
		if !ValidSort(s) {
			t.Errorf("ValidSort(%q) = false, want true", s)
		}
	}
	if ValidSort("sale") {
		t.Error("ValidSort(sale) = true, want false")
	}
}
