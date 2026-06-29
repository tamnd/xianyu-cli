package xianyu_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/any-cli/kit/errs"
	. "github.com/tamnd/xianyu-cli/xianyu"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClient(cfg), srv
}

func isRateLimited(err error) bool { return errs.KindOf(err) == errs.KindRateLimited }
func isNotFound(err error) bool    { return errs.KindOf(err) == errs.KindNotFound }
func isUsage(err error) bool       { return errs.KindOf(err) == errs.KindUsage }

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL == "" {
		t.Error("BaseURL is empty")
	}
	if cfg.UserAgent == "" {
		t.Error("UserAgent is empty")
	}
	if cfg.Rate <= 0 {
		t.Error("Rate must be > 0")
	}
}

func TestNewClient(t *testing.T) {
	if c := NewClient(DefaultConfig()); c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestIsBlocked_403(t *testing.T) {
	if !IsBlocked(403, []byte("forbidden")) {
		t.Error("expected blocked on 403")
	}
}

func TestIsBlocked_JSChallenge(t *testing.T) {
	body := []byte("var __jsl_clearance_s = 'test';")
	if !IsBlocked(200, body) {
		t.Error("expected blocked on __jsl_clearance_s body")
	}
}

func TestIsBlocked_Normal(t *testing.T) {
	// Large body with __NEXT_DATA__ is not blocked
	body := make([]byte, 5000)
	copy(body, []byte(`<script id="__NEXT_DATA__" type="application/json">{}</script>`))
	if IsBlocked(200, body) {
		t.Error("expected NOT blocked on normal body with __NEXT_DATA__")
	}
}

const searchHTML = `<!DOCTYPE html>
<html><body>
<script id="__NEXT_DATA__" type="application/json">
{"props":{"pageProps":{"data":{"resultList":[
  {"data":{"itemId":"111","title":"iPhone 14","price":"4500","area":"北京","picUrl":"//img.goofish.com/a.jpg","sellerNick":"seller1","soldOut":false}},
  {"data":{"itemId":"222","title":"MacBook Pro","price":"8800","area":"上海","picUrl":"//img.goofish.com/b.jpg","sellerNick":"seller2","soldOut":false}}
]}}}}
</script>
</body></html>`

func TestSearch_OK(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(searchHTML))
	})
	defer srv.Close()

	listings, err := c.Search(context.Background(), "iphone", "new", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(listings) != 2 {
		t.Errorf("got %d listings, want 2", len(listings))
	}
	if listings[0].ID != "111" {
		t.Errorf("listings[0].ID = %q, want 111", listings[0].ID)
	}
	if listings[0].Price != 4500 {
		t.Errorf("listings[0].Price = %v, want 4500", listings[0].Price)
	}
	if listings[0].Thumbnail != "https://img.goofish.com/a.jpg" {
		t.Errorf("Thumbnail not normalized: %q", listings[0].Thumbnail)
	}
}

func TestSearch_Blocked_403(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "iphone", "new", 10)
	if !isRateLimited(err) {
		t.Errorf("expected RateLimited, got: %v", err)
	}
}

func TestSearch_Blocked_JSChallenge(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("var __jsl_clearance_s = 'abc';"))
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "iphone", "new", 10)
	if !isRateLimited(err) {
		t.Errorf("expected RateLimited, got: %v", err)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(searchHTML))
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "", "new", 10)
	if !isUsage(err) {
		t.Errorf("expected Usage error for empty query, got: %v", err)
	}
}

func TestSearch_NoResults(t *testing.T) {
	noResultHTML := `<!DOCTYPE html><html><body>
<script id="__NEXT_DATA__" type="application/json">
{"props":{"pageProps":{"data":{"resultList":[]}}}}
</script></body></html>`
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(noResultHTML))
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "xyzabc123notexist", "new", 10)
	if !isNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

func TestParseListings_OK(t *testing.T) {
	listings, err := ParseListings([]byte(searchHTML))
	if err != nil {
		t.Fatalf("ParseListings: %v", err)
	}
	if len(listings) != 2 {
		t.Errorf("got %d listings, want 2", len(listings))
	}
}
