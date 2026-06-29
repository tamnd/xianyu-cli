package xianyu

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/any-cli/kit/errs"
)

// Listing is one Xianyu/Goofish secondhand listing record.
type Listing struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	Condition string  `json:"condition"`
	Sold      bool    `json:"sold"`
	Location  string  `json:"location"`
	Seller    string  `json:"seller"`
	Thumbnail string  `json:"thumbnail"`
	URL       string  `json:"url"`
}

// sortParam maps CLI sort names to Goofish URL param values.
var sortParam = map[string]string{
	"new":        "default",
	"price-asc":  "price_asc",
	"price-desc": "price_desc",
}

// ValidSort reports whether sort is a known sort value.
func ValidSort(sort string) bool {
	_, ok := sortParam[sort]
	return ok
}

var nextDataRE = regexp.MustCompile(
	`<script\s+id="__NEXT_DATA__"\s+type="application/json">([^<]+)</script>`)

// Search fetches Xianyu search results for query.
func (c *Client) Search(ctx context.Context, query, sort string, limit int) ([]*Listing, error) {
	if query == "" {
		return nil, errs.Usage("query is required")
	}
	sp, ok := sortParam[sort]
	if !ok {
		sp = "default"
	}

	rawURL := fmt.Sprintf(
		"%s/search?keyword=%s&sort=%s",
		c.cfg.BaseURL,
		url.QueryEscape(query),
		sp,
	)

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	listings, err := parseListings(body)
	if err != nil {
		return nil, err
	}
	if len(listings) == 0 {
		return nil, errs.NotFound("no results for %q", query)
	}
	if limit > 0 && len(listings) > limit {
		listings = listings[:limit]
	}
	return listings, nil
}

// parseListings extracts listings from the __NEXT_DATA__ JSON blob.
func parseListings(body []byte) ([]*Listing, error) {
	m := nextDataRE.FindSubmatch(body)
	if m == nil {
		return nil, errs.NotFound("no __NEXT_DATA__ found in response")
	}

	var root map[string]any
	if err := json.Unmarshal(m[1], &root); err != nil {
		return nil, fmt.Errorf("parse __NEXT_DATA__: %w", err)
	}

	// Primary path: props.pageProps.data.resultList
	resultList := walkList(root, "props", "pageProps", "data", "resultList")
	if len(resultList) == 0 {
		// Fallback: props.pageProps.initialData.data.resultList
		resultList = walkList(root, "props", "pageProps", "initialData", "data", "resultList")
	}

	var out []*Listing
	for _, el := range resultList {
		data, ok := el["data"].(map[string]any)
		if !ok {
			data = el
		}
		l := parseListing(data)
		if l != nil && l.ID != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// ParseListings is exported for tests.
func ParseListings(body []byte) ([]*Listing, error) {
	return parseListings(body)
}

func walkList(root map[string]any, keys ...string) []map[string]any {
	var node any = root
	for _, k := range keys {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[k]
	}
	arr, ok := node.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, el := range arr {
		if m, ok := el.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func parseListing(raw map[string]any) *Listing {
	id := rawStr(raw, "itemId")
	if id == "" {
		id = rawStr(raw, "id")
	}
	title := html.UnescapeString(rawStr(raw, "title"))
	price := parsePrice(rawStr(raw, "price"))
	cond := rawStr(raw, "wantlevel")
	sold := rawBool(raw, "soldOut") || rawInt(raw, "status") == 2
	loc := rawStr(raw, "area")
	seller := rawStr(raw, "sellerNick")
	pic := normURL(rawStr(raw, "picUrl"))
	if pic == "" {
		pic = normURL(rawStr(raw, "pic_url"))
	}
	itemURL := "https://www.goofish.com/item?id=" + id
	return &Listing{
		ID:        id,
		Title:     title,
		Price:     price,
		Currency:  "CNY",
		Condition: cond,
		Sold:      sold,
		Location:  loc,
		Seller:    seller,
		Thumbnail: pic,
		URL:       itemURL,
	}
}

func rawStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func rawBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func rawInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

var priceRE = regexp.MustCompile(`[\d.]+`)

// parsePrice parses a price string into float64.
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "¥", "")
	s = strings.ReplaceAll(s, "￥", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if m := priceRE.FindString(s); m != "" {
		f, err := strconv.ParseFloat(m, 64)
		if err == nil {
			return f
		}
	}
	return 0
}
