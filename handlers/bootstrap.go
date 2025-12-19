package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"log"
	"net/http"
	"path"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"

	"github.com/dimaskiddo/play-with-docker/config"
	"github.com/dimaskiddo/play-with-docker/event"
	"github.com/dimaskiddo/play-with-docker/pwd"
	"github.com/dimaskiddo/play-with-docker/pwd/types"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
	oauth2Github "golang.org/x/oauth2/github"
	oauth2Google "golang.org/x/oauth2/google"
	oauth2Microsoft "golang.org/x/oauth2/microsoft"
	"google.golang.org/api/people/v1"
)

var (
	core     pwd.PWDApi
	e        event.EventApi
	landings = map[string][]byte{}
)

//go:embed www/*
var embeddedFiles embed.FS

var staticFiles fs.FS

var latencyHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "pwd_handlers_duration_ms",
	Help:    "How long it took to process a specific handler, in a specific host",
	Buckets: []float64{300, 1200, 5000},
}, []string{"action"})

type HandlerExtender func(h *mux.Router)

func init() {
	prometheus.MustRegister(latencyHistogramVec)
	staticFiles, _ = fs.Sub(embeddedFiles, "www")
}

func HeadersSecurity(w http.ResponseWriter) {
	w.Header().Set("X-Powered-By", "Play-With-Docker (Dimas Restu H)")

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-DNS-Prefetch-Control", "off")

	w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
}

func HeadersNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func MiddlewareHeaderMux(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HeadersSecurity(w)
		HeadersNoCache(w)

		next.ServeHTTP(w, r)
	})
}

func MiddlewareHeaderNegroni(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	HeadersSecurity(w)
	HeadersNoCache(w)

	next(w, r)
}

func Bootstrap(c pwd.PWDApi, ev event.EventApi) {
	core = c
	e = ev
}

func Register(extend HandlerExtender) {
	initPlaygrounds()

	r := mux.NewRouter()
	r.Use(MiddlewareHeaderMux)

	corsRouter := mux.NewRouter()
	corsRouter.Use(MiddlewareHeaderMux)

	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(MiddlewareHeaderNegroni))

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/ping", Ping).Methods("GET")

	r.HandleFunc("/", Landing).Methods("GET")
	r.HandleFunc("/p/{sessionId}", Home).Methods("GET")
	r.HandleFunc("/users/{userId:.{3,}}", GetUser).Methods("GET")
	r.HandleFunc("/oauth/providers", ListProviders).Methods("GET")
	r.HandleFunc("/oauth/providers/{provider}/login", Login).Methods("GET")
	r.HandleFunc("/oauth/providers/{provider}/callback", LoginCallback).Methods("GET")
	r.HandleFunc("/my/playground", GetCurrentPlayground).Methods("GET")
	r.HandleFunc("/playgrounds", NewPlayground).Methods("PUT")
	r.HandleFunc("/playgrounds", ListPlaygrounds).Methods("GET")

	corsRouter.HandleFunc("/", NewSession).Methods("POST")
	corsRouter.HandleFunc("/users/me", LoggedInUser).Methods("GET")
	corsRouter.HandleFunc("/instances/images", GetInstanceImages).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}", GetSession).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/ws/", WSH).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/close", CloseSession).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}", CloseSession).Methods("DELETE")
	corsRouter.HandleFunc("/sessions/{sessionId}/setup", SessionSetup).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances", NewInstance).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/uploads", FileUpload).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", DeleteInstance).Methods("DELETE")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/exec", Exec).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/fstree", fsTree).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/file", file).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/download-key", fileDownloadKey).Methods("GET")

	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/editor", func(rw http.ResponseWriter, r *http.Request) {
		serveAsset(rw, r, "editor.html")
	}).Methods("GET")

	r.PathPrefix("/assets").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		serveAsset(rw, r, r.URL.Path[1:])
	}).Methods("GET")

	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		serveAsset(rw, r, "robots.txt")
	}).Methods("GET")

	r.HandleFunc("/503", func(rw http.ResponseWriter, r *http.Request) {
		serveAsset(rw, r, "503.html")
	}).Methods("GET")

	r.HandleFunc("/ooc", func(rw http.ResponseWriter, r *http.Request) {
		serveAsset(rw, r, "ooc.html")
	}).Methods("GET")

	if extend != nil {
		extend(corsRouter)
	}

	corsHandler := gh.CORS(
		gh.AllowCredentials(),
		gh.AllowedHeaders([]string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization"}),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}),
		gh.AllowedOrigins([]string{"*"}),
		gh.AllowedOriginValidator(func(origin string) bool {
			return true
		}))

	r.PathPrefix("/").Handler(negroni.New(negroni.Wrap(corsHandler(corsRouter))))
	n.UseHandler(r)

	httpServer := http.Server{
		Addr:              "0.0.0.0:" + config.PortNumber,
		Handler:           n,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if config.UseLetsEncrypt {
		domainCache, err := lru.New(5000)
		if err != nil {
			log.Fatalf("Could not Start Domain Cache. Got: %v", err)
		}

		certManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			HostPolicy: func(ctx context.Context, host string) error {
				if _, found := domainCache.Get(host); !found {
					if playground := core.PlaygroundFindByDomain(host); playground == nil {
						return fmt.Errorf("Playground for Domain %s was Not Found", host)
					}
					domainCache.Add(host, true)
				}
				return nil
			},
			Cache: autocert.DirCache(config.LetsEncryptCertsDir),
		}

		httpServer.TLSConfig = &tls.Config{
			GetCertificate: certManager.GetCertificate,
		}

		go func() {
			rr := mux.NewRouter()
			rr.Use(MiddlewareHeaderMux)

			rr.Handle("/metrics", promhttp.Handler())
			rr.HandleFunc("/ping", Ping).Methods("GET")

			rr.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
				target := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
				if len(r.URL.RawQuery) > 0 {
					target += "?" + r.URL.RawQuery
				}

				http.Redirect(rw, r, target, http.StatusMovedPermanently)
			})

			nr := negroni.Classic()
			nr.Use(negroni.HandlerFunc(MiddlewareHeaderNegroni))

			nr.UseHandler(rr)

			redirectServer := http.Server{
				Addr:              "0.0.0.0:3001",
				Handler:           certManager.HTTPHandler(nr),
				IdleTimeout:       30 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
			}

			log.Fatal(redirectServer.ListenAndServe())
		}()

		log.Println("Listening on Port " + config.PortNumber)
		log.Fatal(httpServer.ListenAndServeTLS("", ""))
	} else {
		log.Println("Listening on Port " + config.PortNumber)
		log.Fatal(httpServer.ListenAndServe())
	}
}

func serveAsset(w http.ResponseWriter, r *http.Request, name string) {
	a, err := fs.ReadFile(staticFiles, name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(a))
}

func initPlaygrounds() {
	pgs, err := core.PlaygroundList()
	if err != nil {
		log.Fatal("Error Getting Playgrounds for Initialization")
	}

	for _, p := range pgs {
		initAssets(p)
		initOauthProviders(p)
	}
}

func initAssets(p *types.Playground) {
	if p.AssetsDir == "" {
		p.AssetsDir = "default"
	}

	lpath := path.Join(p.AssetsDir, "landing.html")
	landing, err := fs.ReadFile(staticFiles, lpath)
	if err != nil {
		log.Printf("Could not Load %v: %v", lpath, err)
		return
	}

	var b bytes.Buffer

	t := template.New("landing.html").Delims("[[", "]]")
	t, err = t.Parse(string(landing))
	if err != nil {
		log.Fatalf("Error parsing template %v", err)
	}

	if err := t.Execute(&b, struct{ SegmentId string }{config.SegmentId}); err != nil {
		log.Fatalf("Error executing template %v", err)
	}

	landingBytes, err := io.ReadAll(&b)
	if err != nil {
		log.Fatalf("Error reading template bytes %v", err)
	}

	landings[p.Id] = landingBytes
}

func initOauthProviders(p *types.Playground) {
	config.Providers[p.Id] = map[string]*oauth2.Config{}

	if p.DockerClientID != "" && p.DockerClientSecret != "" {
		dockerEndpoint := getDockerEndpoint(p)

		conf := &oauth2.Config{
			ClientID:     p.DockerClientID,
			ClientSecret: p.DockerClientSecret,
			Scopes:       []string{"openid", "full_access:account"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%s/authorize", dockerEndpoint),
				TokenURL: fmt.Sprintf("https://%s/oauth/token", dockerEndpoint),
			},
		}

		config.Providers[p.Id]["docker"] = conf
	}

	if p.GithubClientID != "" && p.GithubClientSecret != "" {
		conf := &oauth2.Config{
			ClientID:     p.GithubClientID,
			ClientSecret: p.GithubClientSecret,
			Scopes:       []string{"user:email"},
			Endpoint:     oauth2Github.Endpoint,
		}

		config.Providers[p.Id]["github"] = conf
	}

	if p.GoogleClientID != "" && p.GoogleClientSecret != "" {
		conf := &oauth2.Config{
			ClientID:     p.GoogleClientID,
			ClientSecret: p.GoogleClientSecret,
			Scopes:       []string{people.UserinfoEmailScope, people.UserinfoProfileScope},
			Endpoint:     oauth2Google.Endpoint,
		}

		config.Providers[p.Id]["google"] = conf
	}

	if p.AzureClientID != "" && p.AzureClientSecret != "" {
		conf := &oauth2.Config{
			ClientID:     p.AzureClientID,
			ClientSecret: p.AzureClientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     oauth2Microsoft.AzureADEndpoint(p.AzureTenantID),
		}

		config.Providers[p.Id]["azure"] = conf
	}

	if p.OIDCClientID != "" && p.OIDCClientSecret != "" && p.OIDCEndpoint != "" {
		conf := &oauth2.Config{
			ClientID:     p.OIDCClientID,
			ClientSecret: p.OIDCClientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%s/login/oauth/authorize", p.OIDCEndpoint),
				TokenURL: fmt.Sprintf("https://%s/login/oauth/access_token", p.OIDCEndpoint),
			},
		}

		config.Providers[p.Id]["oidc"] = conf
	}
}
