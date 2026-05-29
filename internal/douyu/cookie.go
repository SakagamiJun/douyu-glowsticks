package douyu

import (
	"sort"

	"douyu-glowsticks/internal/config"

	http "github.com/bogdanfinn/fhttp"
	"github.com/chromedp/cdproto/network"
)

// MergeRawCookies 合并旧的 Cookie 结构数组和新的 Cookie 数组，使用复合键避免覆盖跨域 Cookie
func MergeRawCookies(oldCookies []config.Cookie, newCookies []config.Cookie) ([]config.Cookie, bool) {
	if len(newCookies) == 0 {
		return oldCookies, false
	}
	cookieMap := make(map[string]config.Cookie)
	for _, c := range oldCookies {
		key := c.Name + "|" + c.Domain + "|" + c.Path
		cookieMap[key] = c
	}

	changed := false
	for _, nc := range newCookies {
		key := nc.Name + "|" + nc.Domain + "|" + nc.Path
		if existing, ok := cookieMap[key]; !ok || existing.Value != nc.Value {
			cookieMap[key] = nc
			changed = true
		}
	}

	if !changed {
		return oldCookies, false
	}

	var out []config.Cookie
	var keys []string
	for k := range cookieMap {
		keys = append(keys, k)
	}
	// Sort to keep order consistent
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, cookieMap[k])
	}
	return out, true
}

// MergeHttpCookies 合并 http.Cookie 到 config.Cookie
func MergeHttpCookies(oldCookies []config.Cookie, newHttpCookies []*http.Cookie, fallbackDomain string) ([]config.Cookie, bool) {
	var newCookies []config.Cookie
	for _, hc := range newHttpCookies {
		domain := hc.Domain
		if domain == "" {
			domain = fallbackDomain
		}
		path := hc.Path
		if path == "" {
			path = "/"
		}
		newCookies = append(newCookies, config.Cookie{
			Name:   hc.Name,
			Value:  hc.Value,
			Domain: domain,
			Path:   path,
		})
	}
	return MergeRawCookies(oldCookies, newCookies)
}

func networkCookiesToConfig(cookies []*network.Cookie) []config.Cookie {
	var out []config.Cookie
	for _, c := range cookies {
		out = append(out, config.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	return out
}
