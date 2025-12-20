package handlers

import (
	"compress/gzip"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/dimaskiddo/play-with-docker/config"
	"github.com/urfave/negroni"
)

var gzipHandler, _ = gziphandler.NewGzipLevelHandler(gzip.BestSpeed)

func headerRealIP(r *http.Request) {
	_, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		port = "0"
	}

	rip := ""
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		rip = cfip
	} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		rip = xrip
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		rip = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	if rip != "" {
		r.RemoteAddr = net.JoinHostPort(rip, port)
	}
}

func headersSecurity(w http.ResponseWriter) {
	w.Header().Set("X-Powered-By", "Play-With-Docker (Dimas Restu H)")

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-DNS-Prefetch-Control", "off")

	w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")

	cdnList := []string{
		"https://apis.google.com",
		"https://code.jquery.com",
		"https://unpkg.com",
		"https://cdn.rawgit.com",
		"https://cdn.jsdelivr.net",
		"https://cdnjs.cloudflare.com",
		"https://cloudflareinsights.com",
		"https://*.cloudflareinsights.com",
		"https://*.cloudflare.com",
		"https://*.googleapis.com",
		"https://*.bootstrapcdn.com",
		"https://*.fastly.net",
		"https://*.fastly.io",
	}

	fontList := []string{
		"https://unpkg.com",
		"https://fonts.gstatic.com",
		"https://*.cloudflare.com",
		"https://*.bootstrapcdn.com",
		"https://*.fastly.net",
		"https://*.fastly.io",
	}

	imgList := []string{
		"data:",
		"https://*.github.io",
		"https://*.githubusercontent.com",
		"https://*.cloudflare.com",
	}

	cspCdn := strings.Join(cdnList, " ")
	cspFont := strings.Join(fontList, " ")
	cspImg := strings.Join(imgList, " ")

	csp := "" +
		"default-src 'self'; " +
		"connect-src 'self' ws: wss: " + cspCdn + "; " +
		"script-src 'self' 'unsafe-inline' 'unsafe-eval' " + cspCdn + "; " +
		"style-src 'self' 'unsafe-inline' " + cspCdn + "; " +
		"font-src 'self' " + cspFont + "; " +
		"img-src 'self' " + cspImg + ";"

	w.Header().Set("Content-Security-Policy", csp)
}

func headersNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func handlerRateLimit(w http.ResponseWriter, r *http.Request) bool {
	skipExtensions := []string{
		".js", ".css", ".html", ".json", ".txt", ".map",
		".png", ".jpg", ".jpeg", ".svg", ".gif", ".ico",
		".woff", ".woff2", ".ttf", ".otf", ".eot",
		".webmanifet",
	}

	path := strings.ToLower(r.URL.Path)
	for _, ext := range skipExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	if !config.RateLimiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		return false
	}

	return true
}

func CustomMiddlewareMux(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerRealIP(r)
		headersSecurity(w)
		headersNoCache(w)

		if !handlerRateLimit(w, r) {
			return
		}

		gzipHandler(next).ServeHTTP(w, r)
	})
}

func CustomMiddlewareNegroni(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	startTime := time.Now()

	headerRealIP(r)
	headersSecurity(w)
	headersNoCache(w)

	if !handlerRateLimit(w, r) {
		return
	}

	gzipHandler(http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
		nw := negroni.NewResponseWriter(rw)
		next(nw, rq)

		rIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			rIP = r.RemoteAddr
		}

		log.Printf("[negroni] %s | %s | %d | %s | %s | %s | %v",
			startTime.Format("2006-01-02 15:04:05"), rIP, nw.Status(),
			r.Host, r.Method, r.URL.Path,
			time.Since(startTime))
	})).ServeHTTP(w, r)
}
