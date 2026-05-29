package douyu

import (
	"strings"
	"testing"
	"time"

	"github.com/SakagamiJun/douyu-glowsticks/internal/config"

	http "github.com/bogdanfinn/fhttp"
)

func TestMergeHttpCookiesPreservesOverlappingDomainCookies(t *testing.T) {
	oldCookies := []config.Cookie{
		{Name: "acf_uid", Value: "old", Domain: ".douyu.com", Path: "/"},
		{Name: "acf_auth", Value: "auth", Domain: ".douyu.com", Path: "/"},
	}
	newCookies := []*http.Cookie{
		{Name: "acf_uid", Value: "new"},
	}

	merged, changed := MergeHttpCookies(oldCookies, newCookies, "www.douyu.com")
	if !changed {
		t.Fatal("MergeHttpCookies() changed = false, want true")
	}

	var acfUID []config.Cookie
	for _, cookie := range merged {
		if cookie.Name == "acf_uid" {
			acfUID = append(acfUID, cookie)
		}
	}
	if len(acfUID) != 2 {
		t.Fatalf("acf_uid cookie count = %d, want 2: %#v", len(acfUID), merged)
	}
	if !hasCookie(merged, "acf_uid", "old", "douyu.com", "/") ||
		!hasCookie(merged, "acf_uid", "new", "www.douyu.com", "/") {
		t.Fatalf("merged cookies = %#v, want both broader-domain and host-domain acf_uid", merged)
	}
}

func TestMergeHttpCookiesDeletesExactExpiredCookie(t *testing.T) {
	oldCookies := []config.Cookie{
		{Name: "acf_uid", Value: "broad", Domain: "douyu.com", Path: "/"},
		{Name: "acf_uid", Value: "old", Domain: "www.douyu.com", Path: "/"},
	}
	newCookies := []*http.Cookie{
		{Name: "acf_uid", Value: "", MaxAge: -1},
	}

	merged, changed := MergeHttpCookies(oldCookies, newCookies, "www.douyu.com")
	if !changed {
		t.Fatal("MergeHttpCookies() changed = false, want true")
	}
	if len(merged) != 1 || !hasCookie(merged, "acf_uid", "broad", "douyu.com", "/") {
		t.Fatalf("merged cookies = %#v, want only exact host-domain cookie removed", merged)
	}
}

func TestMergeHttpCookiesDeletesCookieWithPastExpires(t *testing.T) {
	oldCookies := []config.Cookie{
		{Name: "acf_uid", Value: "old", Domain: "www.douyu.com", Path: "/"},
	}
	newCookies := []*http.Cookie{
		{Name: "acf_uid", Value: "", Expires: time.Now().Add(-time.Minute)},
	}

	merged, changed := MergeHttpCookies(oldCookies, newCookies, "www.douyu.com")
	if !changed {
		t.Fatal("MergeHttpCookies() changed = false, want true")
	}
	if len(merged) != 0 {
		t.Fatalf("merged cookies = %#v, want expired cookie removed", merged)
	}
}

func TestCookiesToHeaderFiltersHostAndKeepsOverlappingNames(t *testing.T) {
	cookies := []config.Cookie{
		{Name: "acf_uid", Value: "old", Domain: "douyu.com", Path: "/"},
		{Name: "acf_uid", Value: "new", Domain: "www.douyu.com", Path: "/"},
		{Name: "qq_session", Value: "qq", Domain: "qq.com", Path: "/"},
	}

	header := cookiesToHeader(cookies, "www.douyu.com", "/")
	if strings.Contains(header, "qq_session=qq") {
		t.Fatalf("Cookie header = %q, want qq.com cookie filtered out", header)
	}
	if strings.Count(header, "acf_uid=") != 2 {
		t.Fatalf("Cookie header = %q, want both matching acf_uid cookies", header)
	}
	if strings.Index(header, "acf_uid=new") > strings.Index(header, "acf_uid=old") {
		t.Fatalf("Cookie header = %q, want host-domain cookie before broader-domain cookie", header)
	}
}

func hasCookie(cookies []config.Cookie, name string, value string, domain string, path string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name &&
			cookie.Value == value &&
			normalizeDomain(cookie.Domain) == normalizeDomain(domain) &&
			normalizePath(cookie.Path) == normalizePath(path) {
			return true
		}
	}
	return false
}
