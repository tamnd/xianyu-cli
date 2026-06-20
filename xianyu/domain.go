package xianyu

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the Xianyu/Goofish driver.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "xianyu",
		Hosts:  []string{Host, "www.goofish.com", "xianyu.taobao.com"},
		Identity: kit.Identity{
			Binary: "xianyu",
			Short:  "Search Xianyu secondhand listings from the command line",
			Long: `xianyu searches www.goofish.com (Xianyu) and formats results as clean records.

Note: Xianyu is Tier C (anti-bot). Requests from non-CN datacenter IPs are
usually blocked by Alibaba's Securitymgr stack. The CLI exits with code 5 when
a block is detected.

Quick start:
  xianyu search iphone
  xianyu search "二手相机" --sort price-asc
  xianyu search macbook -n 20 -o jsonl`,
			Site: Host,
			Repo: "https://github.com/tamnd/xianyu-cli",
		},
	}
}

// Register installs the client factory and all operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "listings",
		List:    true,
		Summary: "Search Xianyu secondhand listings",
		Args:    []kit.Arg{{Name: "query", Help: "search keywords"}},
	}, searchListings)
}

// newClient builds the Xianyu client from the kit-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	dcfg := DefaultConfig()
	if cfg.UserAgent != "" {
		dcfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		dcfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		dcfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		dcfg.Timeout = cfg.Timeout
	}
	return NewClient(dcfg), nil
}

// --- input structs ---

type searchInput struct {
	Query  string  `kit:"arg"          help:"search keywords"`
	Sort   string  `kit:"flag"         help:"sort order: new, price-asc, price-desc"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchListings(ctx context.Context, in searchInput, emit func(*Listing) error) error {
	sort := in.Sort
	if sort == "" {
		sort = "new"
	}
	if !ValidSort(sort) {
		return errs.Usage("unknown sort %q; use new, price-asc, or price-desc", sort)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 30
	}
	listings, err := in.Client.Search(ctx, in.Query, sort, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, l := range listings {
		if err := emit(l); err != nil {
			return err
		}
	}
	return nil
}

// Classify implements the URI resolver interface.
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("xianyu reference is empty")
	}
	return "listing", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "listing" {
		return "", errs.Usage("xianyu has no resource type %q", uriType)
	}
	return "https://www.goofish.com/item?id=" + id, nil
}

// mapErr passes errors through; kit's typed error system handles exit codes.
func mapErr(err error) error {
	return err
}
