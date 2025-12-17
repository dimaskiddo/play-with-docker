package handlers

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func TerminateSession(rw http.ResponseWriter, req *http.Request) {
	ResetCookie(rw, req.Host)

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Logged out successfully"))
}

func ResetCookie(rw http.ResponseWriter, host string) {
	d := getBaseDomain(host)

	if d != "" {
		cookieDomain := &http.Cookie{
			Name:     "id",
			Value:    "",
			Domain:   d,
			Path:     "/",
			MaxAge:   -1,
			SameSite: http.SameSiteDefaultMode,
			Secure:   false,
			HttpOnly: true,
		}
		http.SetCookie(rw, cookieDomain)

		cookieSubDomain := &http.Cookie{
			Name:     "id",
			Value:    "",
			Domain:   "." + d,
			Path:     "/",
			MaxAge:   -1,
			SameSite: http.SameSiteDefaultMode,
			Secure:   false,
			HttpOnly: true,
		}
		http.SetCookie(rw, cookieSubDomain)
	}

	cookieNoDomain := &http.Cookie{
		Name:     "id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteDefaultMode,
		Secure:   false,
		HttpOnly: true,
	}
	http.SetCookie(rw, cookieNoDomain)
}

func getBaseDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}

	if !strings.Contains(domain, "://") {
		domain = "http://" + domain
	}

	u, err := url.Parse(domain)
	if err != nil {
		return ""
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}

	if net.ParseIP(host) == nil {
		etldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(host)
		if err == nil {
			return etldPlusOne
		}

		return host
	}

	return host
}
