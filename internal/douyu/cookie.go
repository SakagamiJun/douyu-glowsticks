package douyu

import (
	"sort"
	"strings"
	"time"

	"douyu-glowsticks/internal/config"

	http "github.com/bogdanfinn/fhttp"
	"github.com/chromedp/cdproto/network"
)

// MergeRawCookies 合并旧的 Cookie 结构数组和新的 Cookie 数组。
// Cookie 在浏览器中以 Name + Domain + Path 为唯一键；不同 Domain 的同名 Cookie 可以合法共存。
func MergeRawCookies(oldCookies []config.Cookie, newCookies []config.Cookie) ([]config.Cookie, bool) {
	if len(newCookies) == 0 {
		return oldCookies, false
	}

	cookieMap := make(map[string]config.Cookie)
	for _, c := range normalizeCookieList(oldCookies) {
		cookieMap[cookieKey(c)] = c
	}

	for _, nc := range newCookies {
		nc = normalizeCookie(nc)
		if nc.Name == "" {
			continue
		}
		cookieMap[cookieKey(nc)] = nc
	}

	merged := make([]config.Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		merged = append(merged, c)
	}
	merged = sortCookieList(merged)

	if cookiesEqual(sortCookieList(normalizeCookieList(oldCookies)), merged) {
		return oldCookies, false
	}

	return merged, true
}

// MergeHttpCookies 合并 http.Cookie 到 config.Cookie，并处理 Set-Cookie 删除语义。
func MergeHttpCookies(oldCookies []config.Cookie, newHttpCookies []*http.Cookie, fallbackDomain string) ([]config.Cookie, bool) {
	if len(newHttpCookies) == 0 {
		return oldCookies, false
	}

	cookieMap := make(map[string]config.Cookie)
	for _, c := range normalizeCookieList(oldCookies) {
		cookieMap[cookieKey(c)] = c
	}

	for _, hc := range newHttpCookies {
		if hc == nil {
			continue
		}

		nc := httpCookieToConfig(hc, fallbackDomain)
		if nc.Name == "" {
			continue
		}
		key := cookieKey(nc)
		if isExpiredHTTPCookie(hc) {
			delete(cookieMap, key)
		} else {
			cookieMap[key] = nc
		}
	}

	merged := make([]config.Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		merged = append(merged, c)
	}
	merged = sortCookieList(merged)

	if cookiesEqual(sortCookieList(normalizeCookieList(oldCookies)), merged) {
		return oldCookies, false
	}

	return merged, true
}

func httpCookieToConfig(hc *http.Cookie, fallbackDomain string) config.Cookie {
	domain := hc.Domain
	if domain == "" {
		domain = fallbackDomain
	}
	return normalizeCookie(config.Cookie{
		Name:   hc.Name,
		Value:  hc.Value,
		Domain: domain,
		Path:   hc.Path,
	})
}

func networkCookiesToConfig(cookies []*network.Cookie) []config.Cookie {
	var out []config.Cookie
	for _, c := range cookies {
		out = append(out, normalizeCookie(config.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		}))
	}
	return out
}

// CookiesToStr 把 Cookie 结构体数组转成用于 HTTP Header 的字符串
func CookiesToStr(cookies []config.Cookie) string {
	return cookiesToHeader(cookies, "", "")
}

func cookiesToHeader(cookies []config.Cookie, host string, requestPath string) string {
	host = normalizeDomain(host)
	requestPath = normalizePath(requestPath)

	selected := make([]config.Cookie, 0, len(cookies))
	for _, c := range normalizeCookieList(cookies) {
		if host != "" && !domainMatchesHost(c.Domain, host) {
			continue
		}
		if !pathMatches(c.Path, requestPath) {
			continue
		}
		selected = append(selected, c)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		leftPath := normalizePath(selected[i].Path)
		rightPath := normalizePath(selected[j].Path)
		if len(leftPath) != len(rightPath) {
			return len(leftPath) > len(rightPath)
		}

		leftDomainScore := domainSpecificity(selected[i].Domain, host)
		rightDomainScore := domainSpecificity(selected[j].Domain, host)
		if leftDomainScore != rightDomainScore {
			return leftDomainScore > rightDomainScore
		}

		return cookieKey(selected[i]) < cookieKey(selected[j])
	})

	parts := make([]string, 0, len(selected))
	for _, c := range selected {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

func normalizeCookieList(cookies []config.Cookie) []config.Cookie {
	cookieMap := make(map[string]config.Cookie)
	for _, c := range cookies {
		c = normalizeCookie(c)
		if c.Name == "" {
			continue
		}
		cookieMap[cookieKey(c)] = c
	}

	out := make([]config.Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		out = append(out, c)
	}
	return sortCookieList(out)
}

func normalizeCookie(c config.Cookie) config.Cookie {
	c.Name = strings.TrimSpace(c.Name)
	c.Domain = normalizeDomain(c.Domain)
	if c.Path == "" {
		c.Path = "/"
	}
	return c
}

func normalizeDomain(domain string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func cookieKey(c config.Cookie) string {
	return c.Name + "|" + normalizeDomain(c.Domain) + "|" + normalizePath(c.Path)
}

func domainMatchesHost(domain string, host string) bool {
	domain = normalizeDomain(domain)
	host = normalizeDomain(host)
	if domain == "" || host == "" {
		return domain == host
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func pathMatches(cookiePath string, requestPath string) bool {
	cookiePath = normalizePath(cookiePath)
	requestPath = normalizePath(requestPath)
	if requestPath == cookiePath {
		return true
	}
	if !strings.HasPrefix(requestPath, cookiePath) {
		return false
	}
	return strings.HasSuffix(cookiePath, "/") || requestPath[len(cookiePath)] == '/'
}

func domainSpecificity(domain string, host string) int {
	domain = normalizeDomain(domain)
	host = normalizeDomain(host)
	if domain == "" {
		return 0
	}
	if domain == host {
		return len(domain) + 1000
	}
	return len(domain)
}

func isExpiredHTTPCookie(cookie *http.Cookie) bool {
	if cookie.MaxAge < 0 {
		return true
	}
	return !cookie.Expires.IsZero() && cookie.Expires.Before(time.Now())
}

func sortCookieList(cookies []config.Cookie) []config.Cookie {
	out := append([]config.Cookie(nil), cookies...)
	sort.Slice(out, func(i, j int) bool {
		return cookieKey(out[i]) < cookieKey(out[j])
	})
	return out
}

func cookiesEqual(a []config.Cookie, b []config.Cookie) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if normalizeCookie(a[i]) != normalizeCookie(b[i]) {
			return false
		}
	}
	return true
}
