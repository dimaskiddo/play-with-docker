package config

import (
	"flag"
	"regexp"

	"github.com/gorilla/securecookie"

	"golang.org/x/oauth2"
)

const (
	PWDPortRegex          = "[0-9]{1,5}"
	PWDDomainRegex        = "[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}"
	AliasNameRegex        = "[0-9|a-z|A-Z|-]*"
	AliasSessionRegex     = "[0-9|a-z|A-Z]{8}"
	AliasGroupRegex       = "(" + AliasNameRegex + ")-(" + AliasSessionRegex + ")"
	PWDHostPortGroupRegex = "^.*ip(" + PWDDomainRegex + ")(?:-?(" + PWDPortRegex + "))?(?:\\..*)?$"
	AliasPortGroupRegex   = "^.*pwd" + AliasGroupRegex + "(?:-?(" + PWDPortRegex + "))?\\..*$"
)

var (
	NameFilter  = regexp.MustCompile(PWDHostPortGroupRegex)
	AliasFilter = regexp.MustCompile(AliasPortGroupRegex)
)

var (
	PortNumber, PlaygroundDomain, PWDContainerName, L2ContainerName, L2RouterIP, L2Subdomain, L2SSHPort,
	SessionsFile, SessionDuration, HashKey, CookieHashKey, CookieBlockKey, SSHKeyPath,
	LetsEncryptCertsDir, DataDirHost, DataDirUser, AdminToken, SegmentId string
)

var (
	// Unsafe enables a number of unsafe features when set. It is principally
	// intended to be used in development. For example, it allows the caller to
	// specify the Docker networks to join.
	UseLetsEncrypt, NoWindows, ForceTLS, ExternalDindVolume, Unsafe bool
	DefaultDINDImage, ExternalDindVolumeSize                        string
	DefaultLimitCPUCore, DefaultMaxCPUCore, MaxLoadAvg              float64
	DefaultLimitMemory, DefaultMaxMemory                            int64
	SecureCookie                                                    *securecookie.SecureCookie
)

var (
	DockerClientID, DockerClientSecret              string
	GithubClientID, GithubClientSecret              string
	GoogleClientID, GoogleClientSecret              string
	AzureClientID, AzureClientSecret, AzureTenantID string
	OIDCClientID, OIDCClientSecret, OIDCEndpoint    string
)

var Providers = map[string]map[string]*oauth2.Config{}

func ParseFlags() {
	flag.StringVar(&PortNumber, "port", GetEnvString("PWD_PORT", "3000"), "Play With Docker Port")
	flag.StringVar(&PlaygroundDomain, "domain", GetEnvString("PWD_DOMAIN", "localhost"), "Play With Docker Domain")

	flag.StringVar(&DockerClientID, "oauth-docker-client-id", GetEnvString("PWD_OAUTH_DOCKER_CLIENT_ID", ""), "OAuth Docker Provider Client ID")
	flag.StringVar(&DockerClientSecret, "oauth-docker-client-secret", GetEnvString("PWD_OAUTH_DOCKER_CLIENT_SECRET", ""), "OAuth Docker Provider Client Secret")

	flag.StringVar(&GithubClientID, "oauth-github-client-id", GetEnvString("PWD_OAUTH_GITHUB_CLIENT_ID", ""), "OAuth GitHub Provider Client ID")
	flag.StringVar(&GithubClientSecret, "oauth-github-client-secret", GetEnvString("PWD_OAUTH_GITHUB_CLIENT_SECRET", ""), "OAuth GitHub Provider Client Secret")

	flag.StringVar(&GoogleClientID, "oauth-google-client-id", GetEnvString("PWD_OAUTH_GOOGLE_CLIENT_ID", ""), "OAuth Google Provider Client ID")
	flag.StringVar(&GoogleClientSecret, "oauth-google-client-secret", GetEnvString("PWD_OAUTH_GOOGLE_CLIENT_SECRET", ""), "OAuth Google Provider Client Secret")

	flag.StringVar(&AzureClientID, "oauth-azure-client-id", GetEnvString("PWD_OAUTH_AZURE_CLIENT_ID", ""), "OAuth Azure Provider Client ID")
	flag.StringVar(&AzureClientSecret, "oauth-azure-client-secret", GetEnvString("PWD_OAUTH_AZURE_CLIENT_SECRET", ""), "OAuth Azure Provider Client Secret")
	flag.StringVar(&AzureTenantID, "oauth-azure-tenant-id", GetEnvString("PWD_OAUTH_AZURE_TENANT_ID", "common"), "OAuth Azure Provider Tenant ID")

	flag.StringVar(&OIDCClientID, "oauth-oidc-client-id", GetEnvString("PWD_OAUTH_OIDC_CLIENT_ID", ""), "OAuth OIDC Provider Client ID")
	flag.StringVar(&OIDCClientSecret, "oauth-oidc-client-secret", GetEnvString("PWD_OAUTH_OIDC_CLIENT_SECRET", ""), "OAuth OIDC Provider Client Secret")
	flag.StringVar(&OIDCEndpoint, "oauth-oidc-endpoint", GetEnvString("PWD_OAUTH_OIDC_ENDPOINT", ""), "OAuth OIDC Provider Endpoint")

	flag.StringVar(&PWDContainerName, "name", GetEnvString("PWD_CONTAINER_NAME", "play-with-docker"), "Play With Docker Container Name")
	flag.StringVar(&L2ContainerName, "l2-name", GetEnvString("PWD_L2_CONTAINER_NAME", "play-with-docker-router"), "L2 Router Container Name")
	flag.StringVar(&L2RouterIP, "l2-ip", GetEnvString("PWD_L2_ROUTER_IP", ""), "L2 Router IP address for Ping Response")
	flag.StringVar(&L2Subdomain, "l2-subdomain", GetEnvString("PWD_L2_SUBDOMAIN", "apps"), "L2 Router Subdomain for Ingress")
	flag.StringVar(&L2SSHPort, "l2-ssh-port", GetEnvString("PWD_L2_SSH_PORT", "2222"), "L2 Router Custom SSH Port")

	flag.StringVar(&SessionsFile, "session-file", GetEnvString("PWD_SESSION_FILE", "./sessions/session"), "Path Where Session File will be Stored")
	flag.StringVar(&SessionDuration, "max-session-duration", GetEnvString("PWD_MAX_SESSION_DURATION", "4h"), "Maximum Session Duration Per-User")

	flag.StringVar(&DefaultDINDImage, "default-dind-image", GetEnvString("PWD_DEFAULT_DIND_IMAGE", "franela/dind:latest"), "Default Docker-in-Docker Image")

	flag.Float64Var(&DefaultLimitCPUCore, "default-limit-cpu", GetEnvFloat64("PWD_DEFAULT_LIMIT_CPU", 1.0), "Default Resource Limit for CPU Core")
	flag.Int64Var(&DefaultLimitMemory, "default-limit-memory", GetEnvInt64("PWD_DEFAULT_LIMIT_MEMORY", 2048), "Default Resource Limit for Memory")

	flag.Float64Var(&DefaultMaxCPUCore, "default-max-cpu", GetEnvFloat64("PWD_DEFAULT_MAX_CPU", 4.0), "Default Maximum Limit for CPU Core")
	flag.Int64Var(&DefaultMaxMemory, "default-max-memory", GetEnvInt64("PWD_DEFAULT_MAX_MEMORY", 8192), "Default Maximum Limit for Memory")

	flag.Float64Var(&MaxLoadAvg, "max-load-avg", GetEnvFloat64("PWD_MAX_LOAD_AVG", 100), "Maximum Allowed Load Average Before Failing Ping Requests")

	flag.StringVar(&HashKey, "cookies-secret", GetEnvString("PWD_COOKIES_SECRET", "play-with-docker-cookies"), "Cookies Secret")
	flag.StringVar(&CookieHashKey, "cookies-key-hash", GetEnvString("PWD_COOKIES_KEY_HASH", ""), "Cookies Validation Hash Key")
	flag.StringVar(&CookieBlockKey, "cookies-key-encrypt", GetEnvString("PWD_COOKIES_KEY_ENCRYPT", ""), "Cookies Encryption Key")

	flag.StringVar(&SSHKeyPath, "ssh-key-file", GetEnvString("PWD_SSH_KEY_FILE", "/etc/ssh/ssh_host_rsa_key"), "SSH Private Key to Use")

	flag.BoolVar(&UseLetsEncrypt, "letsencrypt-enable", GetEnvBool("PWD_LETS_ENCRYPT_ENABLE", false), "Enabled Let's Encrypt for TLS Certificates")
	flag.StringVar(&LetsEncryptCertsDir, "letsencrypt-certs-dir", GetEnvString("PWD_LETS_ENCRYPT_CERTS_DIR", "./certs"), "Path Where Let's Encrypt Certificates Will be Stored")

	flag.BoolVar(&NoWindows, "windows-disable", GetEnvBool("PWD_WINDOWS_DISABLE", true), "Disable Windows Instances Support")

	flag.BoolVar(&ForceTLS, "docker-use-tls", GetEnvBool("PWD_DOCKER_USE_TLS", false), "Force TLS Connection to Docker Daemons")
	flag.BoolVar(&ExternalDindVolume, "docker-use-ext-volume", GetEnvBool("PWD_DOCKER_USE_EXTERNAL_VOLUME", false), "Use DIND External Volume Through XFS Volume Driver")
	flag.StringVar(&ExternalDindVolumeSize, "docker-ext-volume-size", GetEnvString("PWD_DOCKER_EXTERNAL_VOLUME_SIZE", ""), "DIND External Volume Size")

	flag.StringVar(&DataDirUser, "data-dir", GetEnvString("PWD_DATA_DIR", "./data"), "Data Directory Inside Container to Store User Persistent Data")

	flag.StringVar(&AdminToken, "admin-token", GetEnvString("PWD_ADMIN_TOKEN", ""), "Token to Validate Admin User for Admin Endpoints")
	flag.StringVar(&SegmentId, "segment-id", GetEnvString("PWD_SEGMENT_ID", ""), "Segment ID to Post Metrics")

	flag.BoolVar(&Unsafe, "unsafe-mode", GetEnvBool("PWD_UNSAFE_MODE", false), "Operate in UnSafe Mode")
	flag.Parse()

	DataDirHost = GetEnvString("PWD_DATA_DIR_HOST", DataDirUser)

	SecureCookie = securecookie.New([]byte(CookieHashKey), []byte(CookieBlockKey))
}
