package handlers

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dimaskiddo/play-with-docker/config"
	"golang.org/x/net/publicsuffix"
)

type CookieID struct {
	Id         string `json:"id"`
	UserName   string `json:"user_name"`
	UserAvatar string `json:"user_avatar"`
	ProviderId string `json:"provider_id"`
}

func (c *CookieID) SetCookie(rw http.ResponseWriter, host string) error {
	if encoded, err := config.SecureCookie.Encode("id", c); err == nil {
		cookie := &http.Cookie{
			Name:     "id",
			Value:    encoded,
			Domain:   host,
			Path:     "/",
			SameSite: http.SameSiteDefaultMode,
			Secure:   false,
			HttpOnly: true,
		}
		http.SetCookie(rw, cookie)
	} else {
		return err
	}
	return nil
}

func ReadCookie(r *http.Request) (*CookieID, error) {
	if cookie, err := r.Cookie("id"); err == nil {
		value := &CookieID{}
		if err = config.SecureCookie.Decode("id", cookie.Value, &value); err == nil {
			return value, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
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
