package skipper

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	stdlog "log"

	ot "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/zalando/skipper/circuit"
	"github.com/zalando/skipper/dataclients/kubernetes"
	"github.com/zalando/skipper/dataclients/routestring"
	"github.com/zalando/skipper/eskip"
	"github.com/zalando/skipper/eskipfile"
	"github.com/zalando/skipper/etcd"
	"github.com/zalando/skipper/filters"
	"github.com/zalando/skipper/filters/apiusagemonitoring"
	"github.com/zalando/skipper/filters/auth"
	"github.com/zalando/skipper/filters/builtin"
	"github.com/zalando/skipper/filters/fadein"
	logfilter "github.com/zalando/skipper/filters/log"
	ratelimitfilters "github.com/zalando/skipper/filters/ratelimit"
	"github.com/zalando/skipper/innkeeper"
	"github.com/zalando/skipper/loadbalancer"
	"github.com/zalando/skipper/logging"
	"github.com/zalando/skipper/metrics"
	skpnet "github.com/zalando/skipper/net"
	pauth "github.com/zalando/skipper/predicates/auth"
	"github.com/zalando/skipper/predicates/cookie"
	"github.com/zalando/skipper/predicates/cron"
	"github.com/zalando/skipper/predicates/forwarded"
	"github.com/zalando/skipper/predicates/host"
	"github.com/zalando/skipper/predicates/interval"
	"github.com/zalando/skipper/predicates/methods"
	"github.com/zalando/skipper/predicates/primitive"
	"github.com/zalando/skipper/predicates/query"
	"github.com/zalando/skipper/predicates/source"
	"github.com/zalando/skipper/predicates/tee"
	"github.com/zalando/skipper/predicates/traffic"
	"github.com/zalando/skipper/proxy"
	"github.com/zalando/skipper/queuelistener"
	"github.com/zalando/skipper/ratelimit"
	"github.com/zalando/skipper/routing"
	"github.com/zalando/skipper/scheduler"
	"github.com/zalando/skipper/secrets"
	"github.com/zalando/skipper/swarm"
	"github.com/zalando/skipper/tracing"
)

const (
	defaultSourcePollTimeout   = 30 * time.Millisecond
	defaultRoutingUpdateBuffer = 1 << 5
)

const DefaultPluginDir = "./plugins"

type testOptions struct {
	redisConnMetricsInterval time.Duration
}

// Options to start skipper.
type Options struct {
	// WaitForHealthcheckInterval sets the time that skipper waits
	// for the loadbalancer in front to become unhealthy. Defaults
	// to 0.
	WaitForHealthcheckInterval time.Duration

	// StatusChecks is an experimental feature. It defines a
	// comma separated list of HTTP URLs to do GET requests to,
	// that have to return 200 before skipper becomes ready
	StatusChecks []string

	// WhitelistedHealthcheckCIDR appends the whitelisted IP Range to the inernalIPS range for healthcheck purposes
	WhitelistedHealthCheckCIDR []string

	// Network address that skipper should listen on.
	Address string

	// EnableTCPQueue enables controlling the
	// concurrently processed requests at the TCP listener.
	EnableTCPQueue bool

	// ExpectedBytesPerRequest is used by the TCP LIFO listener.
	// It defines the expected average memory required to process an incoming
	// request. It is used only when MaxTCPListenerConcurrency is not defined.
	// It is used together with the memory limit defined in:
	// /sys/fs/cgroup/memory/memory.limit_in_bytes.
	//
	// See also: https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
	ExpectedBytesPerRequest int

	// MaxTCPListenerConcurrency is used by the TCP LIFO listener.
	// It defines the max number of concurrently accepted connections, excluding
	// the pending ones in the queue.
	//
	// When undefined and the EnableTCPQueue is true,
	MaxTCPListenerConcurrency int

	// MaxTCPListenerQueue is used by the TCP LIFO listener.
	// If defines the maximum number of pending connection waiting in the queue.
	MaxTCPListenerQueue int

	// List of custom filter specifications.
	CustomFilters []filters.Spec

	// Urls of nodes in an etcd cluster, storing route definitions.
	EtcdUrls []string

	// Path prefix for skipper related data in the etcd storage.
	EtcdPrefix string

	// Timeout used for a single request when querying for updates
	// in etcd. This is independent of, and an addition to,
	// SourcePollTimeout. When not set, the internally defined 1s
	// is used.
	EtcdWaitTimeout time.Duration

	// Skip TLS certificate check for etcd connections.
	EtcdInsecure bool

	// If set this value is used as Bearer token for etcd OAuth authorization.
	EtcdOAuthToken string

	// If set this value is used as username for etcd basic authorization.
	EtcdUsername string

	// If set this value is used as password for etcd basic authorization.
	EtcdPassword string

	// If set enables skipper to generate based on ingress resources in kubernetes cluster
	Kubernetes bool

	// If set makes skipper authenticate with the kubernetes API server with service account assigned to the
	// skipper POD.
	// If omitted skipper will rely on kubectl proxy to authenticate with API server
	KubernetesInCluster bool

	// Kubernetes API base URL. Only makes sense if KubernetesInCluster is set to false. If omitted and
	// skipper is not running in-cluster, the default API URL will be used.
	KubernetesURL string

	// KubernetesHealthcheck, when Kubernetes ingress is set, indicates
	// whether an automatic healthcheck route should be generated. The
	// generated route will report healthyness when the Kubernetes API
	// calls are successful. The healthcheck endpoint is accessible from
	// internal IPs, with the path /kube-system/healthz.
	KubernetesHealthcheck bool

	// KubernetesHTTPSRedirect, when Kubernetes ingress is set, indicates
	// whether an automatic redirect route should be generated to redirect
	// HTTP requests to their HTTPS equivalent. The generated route will
	// match requests with the X-Forwarded-Proto and X-Forwarded-Port,
	// expected to be set by the load-balancer.
	KubernetesHTTPSRedirect bool

	// KubernetesHTTPSRedirectCode overrides the default redirect code (308)
	// when used together with -kubernetes-https-redirect.
	KubernetesHTTPSRedirectCode int

	// KubernetesIngressV1 will switch the dataclient to read ingress v1 resources, instead of v1beta1
	KubernetesIngressV1 bool

	// KubernetesIngressClass is a regular expression, that will make
	// skipper load only the ingress resources that have a matching
	// kubernetes.io/ingress.class annotation. For backwards compatibility,
	// the ingresses without an annotation, or an empty annotation, will
	// be loaded, too.
	KubernetesIngressClass string

	// KubernetesRouteGroupClass is a regular expression, that will make skipper
	// load only the RouteGroup resources that have a matching
	// zalando.org/routegroup.class annotation. Any RouteGroups without the
	// annotation, or which an empty annotation, will be loaded too.
	KubernetesRouteGroupClass string

	// PathMode controls the default interpretation of ingress paths in cases
	// when the ingress doesn't specify it with an annotation.
	KubernetesPathMode kubernetes.PathMode

	// KubernetesNamespace is used to switch between monitoring ingresses in the cluster-scope or limit
	// the ingresses to only those in the specified namespace. Defaults to "" which means monitor ingresses
	// in the cluster-scope.
	KubernetesNamespace string

	// *DEPRECATED* KubernetesEnableEastWest enables cluster internal service to service communication, aka east-west traffic
	KubernetesEnableEastWest bool

	// *DEPRECATED* KubernetesEastWestDomain sets the cluster internal domain used to create additional routes in skipper, defaults to skipper.cluster.local
	KubernetesEastWestDomain string

	// KubernetesEastWestRangeDomains set the the cluster internal domains for
	// east west traffic. Identified routes to such domains will include
	// the KubernetesEastWestRangePredicates.
	KubernetesEastWestRangeDomains []string

	// KubernetesEastWestRangePredicates set the Predicates that will be
	// appended to routes identified as to KubernetesEastWestRangeDomains.
	KubernetesEastWestRangePredicates []*eskip.Predicate

	// KubernetesOnlyAllowedExternalNames will enable validation of ingress external names and route groups network
	// backend addresses, explicit LB endpoints validation agains the list of patterns in
	// AllowedExternalNames.
	KubernetesOnlyAllowedExternalNames bool

	// KubernetesAllowedExternalNames contains regexp patterns of those domain names that are allowed to be
	// used with external name services (type=ExternalName).
	KubernetesAllowedExternalNames []*regexp.Regexp

	// *DEPRECATED* API endpoint of the Innkeeper service, storing route definitions.
	InnkeeperUrl string

	// *DEPRECATED* Fixed token for innkeeper authentication. (Used mainly in
	// development environments.)
	InnkeeperAuthToken string

	// *DEPRECATED* Filters to be prepended to each route loaded from Innkeeper.
	InnkeeperPreRouteFilters string

	// *DEPRECATED* Filters to be appended to each route loaded from Innkeeper.
	InnkeeperPostRouteFilters string

	// *DEPRECATED* Skip TLS certificate check for Innkeeper connections.
	InnkeeperInsecure bool

	// *DEPRECATED* OAuth2 URL for Innkeeper authentication.
	OAuthUrl string

	// *DEPRECATED* Directory where oauth credentials are stored, with file names:
	// client.json and user.json.
	OAuthCredentialsDir string

	// *DEPRECATED* The whitespace separated list of OAuth2 scopes.
	OAuthScope string

	// File containing static route definitions. Multiple may be given comma separated.
	RoutesFile string

	// File containing route definitions with file watch enabled.
	// Multiple may be given comma separated. (For the skipper
	// command this option is used when starting it with the -routes-file flag.)
	WatchRoutesFile string

	// RouteURLs are URLs pointing to route definitions, in eskip format, with change watching enabled.
	RoutesURLs []string

	// InlineRoutes can define routes as eskip text.
	InlineRoutes string

	// Polling timeout of the routing data sources.
	SourcePollTimeout time.Duration

	// DefaultFilters will be applied to all routes automatically.
	DefaultFilters *eskip.DefaultFilters

	// CloneRoute is a PreProcessor, that will be applied to all routes automatically. It
	// will clone all matching routes and apply changes to the
	// cloned routes.
	CloneRoute *eskip.Clone

	// EditRoute will be applied to all routes automatically and
	// will apply changes to all matching routes.
	EditRoute *eskip.Editor

	// Deprecated. See ProxyFlags. When used together with ProxyFlags,
	// the values will be combined with |.
	ProxyOptions proxy.Options

	// Flags controlling the proxy behavior.
	ProxyFlags proxy.Flags

	// Tells the proxy maximum how many idle connections can it keep
	// alive.
	IdleConnectionsPerHost int

	// Defines the time period of how often the idle connections maintained
	// by the proxy are closed.
	CloseIdleConnsPeriod time.Duration

	// Defines ReadTimeoutServer for server http connections.
	ReadTimeoutServer time.Duration

	// Defines ReadHeaderTimeout for server http connections.
	ReadHeaderTimeoutServer time.Duration

	// Defines WriteTimeout for server http connections.
	WriteTimeoutServer time.Duration

	// Defines IdleTimeout for server http connections.
	IdleTimeoutServer time.Duration

	// Defines MaxHeaderBytes for server http connections.
	MaxHeaderBytes int

	// Enable connection state metrics for server http connections.
	EnableConnMetricsServer bool

	// TimeoutBackend sets the TCP client connection timeout for
	// proxy http connections to the backend.
	TimeoutBackend time.Duration

	// ResponseHeaderTimeout sets the HTTP response timeout for
	// proxy http connections to the backend.
	ResponseHeaderTimeoutBackend time.Duration

	// ExpectContinueTimeoutBackend sets the HTTP timeout to expect a
	// response for status Code 100 for proxy http connections to
	// the backend.
	ExpectContinueTimeoutBackend time.Duration

	// KeepAliveBackend sets the TCP keepalive for proxy http
	// connections to the backend.
	KeepAliveBackend time.Duration

	// DualStackBackend sets if the proxy TCP connections to the
	// backend should be dual stack.
	DualStackBackend bool

	// TLSHandshakeTimeoutBackend sets the TLS handshake timeout
	// for proxy connections to the backend.
	TLSHandshakeTimeoutBackend time.Duration

	// MaxIdleConnsBackend sets MaxIdleConns, which limits the
	// number of idle connections to all backends, 0 means no
	// limit.
	MaxIdleConnsBackend int

	// DisableHTTPKeepalives sets DisableKeepAlives, which forces
	// a backend to always create a new connection.
	DisableHTTPKeepalives bool

	// Flag indicating to ignore trailing slashes in paths during route
	// lookup.
	IgnoreTrailingSlash bool

	// Priority routes that are matched against the requests before
	// the standard routes from the data clients.
	PriorityRoutes []proxy.PriorityRoute

	// Specifications of custom, user defined predicates.
	CustomPredicates []routing.PredicateSpec

	// Custom data clients to be used together with the default etcd and Innkeeper.
	CustomDataClients []routing.DataClient

	// CustomHttpHandlerWrap provides ability to wrap http.Handler created by skipper.
	// http.Handler is used for accepting incoming http requests.
	// It allows to add additional logic (for example tracing) by providing a wrapper function
	// which accepts original skipper handler as an argument and returns a wrapped handler
	CustomHttpHandlerWrap func(http.Handler) http.Handler

	// CustomHttpRoundTripperWrap provides ability to wrap http.RoundTripper created by skipper.
	// http.RoundTripper is used for making outgoing requests (backends)
	// It allows to add additional logic (for example tracing) by providing a wrapper function
	// which accepts original skipper http.RoundTripper as an argument and returns a wrapped roundtripper
	CustomHttpRoundTripperWrap func(http.RoundTripper) http.RoundTripper

	// WaitFirstRouteLoad prevents starting the listener before the first batch
	// of routes were applied.
	WaitFirstRouteLoad bool

	// SuppressRouteUpdateLogs indicates to log only summaries of the routing updates
	// instead of full details of the updated/deleted routes.
	SuppressRouteUpdateLogs bool

	// Dev mode. Currently this flag disables prioritization of the
	// consumer side over the feeding side during the routing updates to
	// populate the updated routes faster.
	DevMode bool

	// Network address for the support endpoints
	SupportListener string

	// Deprecated: Network address for the /metrics endpoint
	MetricsListener string

	// Skipper provides a set of metrics with different keys which are exposed via HTTP in JSON
	// You can customize those key names with your own prefix
	MetricsPrefix string

	// EnableProfile exposes profiling information on /profile of the
	// metrics listener.
	EnableProfile bool

	// BlockProfileRate calls runtime.SetBlockProfileRate(BlockProfileRate) if non zero value, deactivate with <0
	BlockProfileRate int

	// MutexProfileFraction calls runtime.SetMutexProfileFraction(MutexProfileFraction) if non zero value, deactivate with <0
	MutexProfileFraction int

	// MemProfileRate calls runtime.SetMemProfileRate(MemProfileRate) if non zero value, deactivate with <0
	MemProfileRate int

	// Flag that enables reporting of the Go garbage collector statistics exported in debug.GCStats
	EnableDebugGcMetrics bool

	// Flag that enables reporting of the Go runtime statistics exported in runtime and specifically runtime.MemStats
	EnableRuntimeMetrics bool

	// If set, detailed response time metrics will be collected
	// for each route, additionally grouped by status and method.
	EnableServeRouteMetrics bool

	// If set, a counter for each route is generated, additionally
	// grouped by status and method. It differs from the automatically
	// generated counter from `EnableServeRouteMetrics` because it will
	// always contain the status and method labels, independently of the
	// `EnableServeMethodMetric` and `EnableServeStatusCodeMetric` flags.
	EnableServeRouteCounter bool

	// If set, detailed response time metrics will be collected
	// for each host, additionally grouped by status and method.
	EnableServeHostMetrics bool

	// If set, a counter for each host is generated, additionally
	// grouped by status and method. It differs from the automatically
	// generated counter from `EnableServeHostMetrics` because it will
	// always contain the status and method labels, independently of the
	// `EnableServeMethodMetric` and `EnableServeStatusCodeMetric` flags.
	EnableServeHostCounter bool

	// If set, the detailed total response time metrics will contain the
	// HTTP method as a domain of the metric. It affects both route and
	// host splitted metrics.
	EnableServeMethodMetric bool

	// If set, the detailed total response time metrics will contain the
	// HTTP Response status code as a domain of the metric. It affects
	// both route and host splitted metrics.
	EnableServeStatusCodeMetric bool

	// If set, detailed response time metrics will be collected
	// for each backend host
	EnableBackendHostMetrics bool

	// EnableAllFiltersMetrics enables collecting combined filter
	// metrics per each route. Without the DisableMetricsCompatibilityDefaults,
	// it is enabled by default.
	EnableAllFiltersMetrics bool

	// EnableCombinedResponseMetrics enables collecting response time
	// metrics combined for every route.
	EnableCombinedResponseMetrics bool

	// EnableRouteResponseMetrics enables collecting response time
	// metrics per each route. Without the DisableMetricsCompatibilityDefaults,
	// it is enabled by default.
	EnableRouteResponseMetrics bool

	// EnableRouteBackendErrorsCounters enables counters for backend
	// errors per each route. Without the DisableMetricsCompatibilityDefaults,
	// it is enabled by default.
	EnableRouteBackendErrorsCounters bool

	// EnableRouteStreamingErrorsCounters enables counters for streaming
	// errors per each route. Without the DisableMetricsCompatibilityDefaults,
	// it is enabled by default.
	EnableRouteStreamingErrorsCounters bool

	// EnableRouteBackendMetrics enables backend response time metrics
	// per each route. Without the DisableMetricsCompatibilityDefaults, it is
	// enabled by default.
	EnableRouteBackendMetrics bool

	// EnableRouteCreationMetrics enables the OriginMarker to track route creation times. Disabled by default
	EnableRouteCreationMetrics bool

	// When set, makes the histograms use an exponentially decaying sample
	// instead of the default uniform one.
	MetricsUseExpDecaySample bool

	// Use custom buckets for prometheus histograms.
	HistogramMetricBuckets []float64

	// The following options, for backwards compatibility, are true
	// by default: EnableAllFiltersMetrics, EnableRouteResponseMetrics,
	// EnableRouteBackendErrorsCounters, EnableRouteStreamingErrorsCounters,
	// EnableRouteBackendMetrics. With this compatibility flag, the default
	// for these options can be set to false.
	DisableMetricsCompatibilityDefaults bool

	// Implementation of a Metrics handler. If provided this is going to be used
	// instead of creating a new one based on the Kind of metrics wanted. This
	// is useful in case you want to report metrics to a custom aggregator.
	MetricsBackend metrics.Metrics

	// Output file for the application log. Default value: /dev/stderr.
	//
	// When /dev/stderr or /dev/stdout is passed in, it will be resolved
	// to os.Stderr or os.Stdout.
	//
	// Warning: passing an arbitrary file will try to open it append
	// on start and use it, or fail on start, but the current
	// implementation doesn't support any more proper handling
	// of temporary failures or log-rolling.
	ApplicationLogOutput string

	// Application log prefix. Default value: "[APP]".
	ApplicationLogPrefix string

	// Enables logs in JSON format
	ApplicationLogJSONEnabled bool

	// ApplicationLogJsonFormatter, when set and JSON logging is enabled, is passed along to to the underlying
	// Logrus logger for application logs. To enable structured logging, use ApplicationLogJSONEnabled.
	ApplicationLogJsonFormatter *log.JSONFormatter

	// Output file for the access log. Default value: /dev/stderr.
	//
	// When /dev/stderr or /dev/stdout is passed in, it will be resolved
	// to os.Stderr or os.Stdout.
	//
	// Warning: passing an arbitrary file will try to open for append
	// it on start and use it, or fail on start, but the current
	// implementation doesn't support any more proper handling
	// of temporary failures or log-rolling.
	AccessLogOutput string

	// Disables the access log.
	AccessLogDisabled bool

	// Enables logs in JSON format
	AccessLogJSONEnabled bool

	// AccessLogStripQuery, when set, causes the query strings stripped
	// from the request URI in the access logs.
	AccessLogStripQuery bool

	// AccessLogJsonFormatter, when set and JSON logging is enabled, is passed along to to the underlying
	// Logrus logger for access logs. To enable structured logging, use AccessLogJSONEnabled.
	AccessLogJsonFormatter *log.JSONFormatter

	DebugListener string

	// Path of certificate(s) when using TLS, mutiple may be given comma separated
	CertPathTLS string
	// Path of key(s) when using TLS, multiple may be given comma separated. For
	// multiple keys, the order must match the one given in CertPathTLS
	KeyPathTLS string

	// TLS Settings for Proxy Server
	ProxyTLS *tls.Config

	// Client TLS to connect to Backends
	ClientTLS *tls.Config

	// TLSMinVersion to set the minimal TLS version for all TLS configurations
	TLSMinVersion uint16

	// Flush interval for upgraded Proxy connections
	BackendFlushInterval time.Duration

	// Experimental feature to handle protocol Upgrades for Websockets, SPDY, etc.
	ExperimentalUpgrade bool

	// ExperimentalUpgradeAudit enables audit log of both the request line
	// and the response messages during web socket upgrades.
	ExperimentalUpgradeAudit bool

	// MaxLoopbacks defines the maximum number of loops that the proxy can execute when the routing table
	// contains loop backends (<loopback>).
	MaxLoopbacks int

	// EnableBreakers enables the usage of the breakers in the route definitions without initializing any
	// by default. It is a shortcut for setting the BreakerSettings to:
	//
	// 	[]circuit.BreakerSettings{{Type: BreakerDisabled}}
	//
	EnableBreakers bool

	// BreakerSettings contain global and host specific settings for the circuit breakers.
	BreakerSettings []circuit.BreakerSettings

	// EnableRatelimiters enables the usage of the ratelimiter in the route definitions without initializing any
	// by default. It is a shortcut for setting the RatelimitSettings to:
	//
	// 	[]ratelimit.Settings{{Type: DisableRatelimit}}
	//
	EnableRatelimiters bool

	// RatelimitSettings contain global and host specific settings for the ratelimiters.
	RatelimitSettings []ratelimit.Settings

	// EnableRouteLIFOMetrics enables metrics for the individual route LIFO queues, if any.
	EnableRouteLIFOMetrics bool

	// OpenTracing enables opentracing
	OpenTracing []string

	// OpenTracingInitialSpan can override the default initial, pre-routing, span name.
	// Default: "ingress".
	OpenTracingInitialSpan string

	// OpenTracingExcludedProxyTags can disable a tag so that it is not recorded. By default every tag is included.
	OpenTracingExcludedProxyTags []string

	// OpenTracingDisableFilterSpans flag is used to disable creation of spans representing request and response filters.
	OpenTracingDisableFilterSpans bool

	// OpenTracingLogFilterLifecycleEvents flag is used to enable/disable the logs for events marking request and
	// response filters' start & end times.
	OpenTracingLogFilterLifecycleEvents bool

	// OpenTracingLogStreamEvents flag is used to enable/disable the logs that marks the
	// times when response headers & payload are streamed to the client
	OpenTracingLogStreamEvents bool

	// OpenTracingBackendNameTag enables an additional tracing tag containing a backend name
	// for a route when it's available (e.g. for RouteGroups)
	OpenTracingBackendNameTag bool

	// PluginDir defines the directory to load plugins from, DEPRECATED, use PluginDirs
	PluginDir string
	// PluginDirs defines the directories to load plugins from
	PluginDirs []string

	// FilterPlugins loads additional filters from modules. The first value in each []string
	// needs to be the plugin name (as on disk, without path, without ".so" suffix). The
	// following values are passed as arguments to the plugin while loading, see also
	// https://opensource.zalando.com/skipper/reference/plugins/
	FilterPlugins [][]string

	// PredicatePlugins loads additional predicates from modules. See above for FilterPlugins
	// what the []string should contain.
	PredicatePlugins [][]string

	// DataClientPlugins loads additional data clients from modules. See above for FilterPlugins
	// what the []string should contain.
	DataClientPlugins [][]string

	// Plugins combine multiple types of the above plugin types in one plugin (where
	// necessary because of shared data between e.g. a filter and a data client).
	Plugins [][]string

	// DefaultHTTPStatus is the HTTP status used when no routes are found
	// for a request.
	DefaultHTTPStatus int

	// EnablePrometheusMetrics enables Prometheus format metrics.
	//
	// This option is *deprecated*. The recommended way to enable prometheus metrics is to
	// use the MetricsFlavours option.
	EnablePrometheusMetrics bool

	// An instance of a Prometheus registry. It allows registering and serving custom metrics when skipper is used as a
	// library.
	// A new registry is created if this option is nil.
	PrometheusRegistry *prometheus.Registry

	// MetricsFlavours sets the metrics storage and exposed format
	// of metrics endpoints.
	MetricsFlavours []string

	// LoadBalancerHealthCheckInterval enables and sets the
	// interval when to schedule health checks for dead or
	// unhealthy routes
	LoadBalancerHealthCheckInterval time.Duration

	// ReverseSourcePredicate enables the automatic use of IP
	// whitelisting in different places to use the reversed way of
	// identifying a client IP within the X-Forwarded-For
	// header. Amazon's ALB for example writes the client IP to
	// the last item of the string list of the X-Forwarded-For
	// header, in this case you want to set this to true.
	ReverseSourcePredicate bool

	// EnableOAuth2GrantFlow, enables OAuth2 Grant Flow filter
	EnableOAuth2GrantFlow bool

	// OAuth2AuthURL, the url to redirect the requests to when login is required.
	OAuth2AuthURL string

	// OAuth2TokenURL, the url where the access code should be exchanged for the
	// access token.
	OAuth2TokenURL string

	// OAuth2RevokeTokenURL, the url where the access and refresh tokens can be
	// revoked during a logout.
	OAuth2RevokeTokenURL string

	// OAuthTokeninfoURL sets the the URL to be queried for
	// information for all auth.NewOAuthTokeninfo*() filters.
	OAuthTokeninfoURL string

	// OAuthTokeninfoTimeout sets timeout duration while calling oauth token service
	OAuthTokeninfoTimeout time.Duration

	// OAuth2SecretFile contains the filename with the encryption key for the
	// authentication cookie and grant flow state stored in Secrets.
	OAuth2SecretFile string

	// OAuth2ClientID, the OAuth2 client id of the current service, used to exchange
	// the access code.
	OAuth2ClientID string

	// OAuth2ClientSecret, the secret associated with the ClientID, used to exchange
	// the access code.
	OAuth2ClientSecret string

	// OAuth2ClientIDFile, the path of the file containing the OAuth2 client id of
	// the current service, used to exchange the access code.
	OAuth2ClientIDFile string

	// OAuth2ClientSecretFile, the path of the file containing the secret associated
	// with the ClientID, used to exchange the access code.
	OAuth2ClientSecretFile string

	// OAuth2CallbackPath contains the path where the OAuth2 callback requests with the
	// authorization code should be redirected to. Defaults to /.well-known/oauth2-callback
	OAuth2CallbackPath string

	// OAuthTokenintrospectionTimeout sets timeout duration while calling oauth tokenintrospection service
	OAuthTokenintrospectionTimeout time.Duration

	// OAuth2AuthURLParameters the additional parameters to send to OAuth2 authorize and token endpoints.
	OAuth2AuthURLParameters map[string]string

	// OAuth2AccessTokenHeaderName the name of the header to which the access token
	// should be assigned after the oauthGrant filter.
	OAuth2AccessTokenHeaderName string

	// OAuth2TokeninfoSubjectKey the key of the subject ID attribute in the
	// tokeninfo map. Used for downstream oidcClaimsQuery compatibility.
	OAuth2TokeninfoSubjectKey string

	// OAuth2TokenCookieName the name of the cookie that Skipper sets after a
	// successful OAuth2 token exchange. Stores the encrypted access token.
	OAuth2TokenCookieName string

	// CompressEncodings, if not empty replace default compression encodings
	CompressEncodings []string

	// OIDCSecretsFile path to the file containing key to encrypt OpenID token
	OIDCSecretsFile string

	// SecretsRegistry to store and load secretsencrypt
	SecretsRegistry *secrets.Registry

	// CredentialsPaths directories or files where credentials are stored one secret per file
	CredentialsPaths []string

	// CredentialsUpdateInterval sets the interval to update secrets
	CredentialsUpdateInterval time.Duration

	// API Monitoring feature is active (feature toggle)
	ApiUsageMonitoringEnable                bool
	ApiUsageMonitoringRealmKeys             string
	ApiUsageMonitoringClientKeys            string
	ApiUsageMonitoringRealmsTrackingPattern string
	// *DEPRECATED* ApiUsageMonitoringDefaultClientTrackingPattern
	ApiUsageMonitoringDefaultClientTrackingPattern string

	// Default filters directory enables default filters mechanism and sets the directory where the filters are located
	DefaultFiltersDir string

	// WebhookTimeout sets timeout duration while calling a custom webhook auth service
	WebhookTimeout time.Duration

	// MaxAuditBody sets the maximum read size of the body read by the audit log filter
	MaxAuditBody int

	// EnableSwarm enables skipper fleet communication, required by e.g.
	// the cluster ratelimiter
	EnableSwarm bool
	// redis based swarm
	SwarmRedisURLs          []string
	SwarmRedisPassword      string
	SwarmRedisHashAlgorithm string
	SwarmRedisDialTimeout   time.Duration
	SwarmRedisReadTimeout   time.Duration
	SwarmRedisWriteTimeout  time.Duration
	SwarmRedisPoolTimeout   time.Duration
	SwarmRedisMinIdleConns  int
	SwarmRedisMaxIdleConns  int
	// swim based swarm
	SwarmKubernetesNamespace          string
	SwarmKubernetesLabelSelectorKey   string
	SwarmKubernetesLabelSelectorValue string
	SwarmPort                         int
	SwarmMaxMessageBuffer             int
	SwarmLeaveTimeout                 time.Duration
	// swim based swarm for local testing
	SwarmStaticSelf  string // 127.0.0.1:9001
	SwarmStaticOther string // 127.0.0.1:9002,127.0.0.1:9003

	// SwarmRegistry specifies an optional callback function that is
	// called after ratelimit registry is initialized
	SwarmRegistry func(*ratelimit.Registry)

	// ClusterRatelimitMaxGroupShards specifies the maximum number of group shards for the clusterRatelimit filter
	ClusterRatelimitMaxGroupShards int

	testOptions
}

type serverErrorLogWriter struct{}

func (*serverErrorLogWriter) Write(p []byte) (int, error) {
	m := string(p)
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF\n") {
		log.Debug(m) // https://github.com/golang/go/issues/26918
	} else if strings.HasPrefix(m, "http: URL query contains semicolon") {
		log.Debug(m) // https://github.com/golang/go/issues/25192
	} else {
		log.Error(m)
	}
	return len(p), nil
}

func newServerErrorLog() *stdlog.Logger {
	return stdlog.New(&serverErrorLogWriter{}, "", 0)
}

func createDataClients(o Options, auth innkeeper.Authentication) ([]routing.DataClient, error) {
	var clients []routing.DataClient

	if o.RoutesFile != "" {
		for _, rf := range strings.Split(o.RoutesFile, ",") {
			f, err := eskipfile.Open(rf)
			if err != nil {
				log.Error("error while opening eskip file", err)
				return nil, err
			}

			clients = append(clients, f)
		}
	}

	if o.WatchRoutesFile != "" {
		for _, rf := range strings.Split(o.WatchRoutesFile, ",") {
			clients = append(clients, eskipfile.Watch(rf))
		}
	}

	if len(o.RoutesURLs) > 0 {
		for _, url := range o.RoutesURLs {
			client, err := eskipfile.RemoteWatch(&eskipfile.RemoteWatchOptions{
				RemoteFile:    url,
				FailOnStartup: true,
				HTTPTimeout:   o.SourcePollTimeout,
			})
			if err != nil {
				log.Errorf("error while loading routes from url %s: %s", url, err)
				return nil, err
			}
			clients = append(clients, client)
		}
	}

	if o.InlineRoutes != "" {
		ir, err := routestring.New(o.InlineRoutes)
		if err != nil {
			log.Error("error while parsing inline routes", err)
			return nil, err
		}

		clients = append(clients, ir)
	}

	if o.InnkeeperUrl != "" {
		ic, err := innkeeper.New(innkeeper.Options{
			Address:          o.InnkeeperUrl,
			Insecure:         o.InnkeeperInsecure,
			Authentication:   auth,
			PreRouteFilters:  o.InnkeeperPreRouteFilters,
			PostRouteFilters: o.InnkeeperPostRouteFilters,
		})

		if err != nil {
			log.Error("error while initializing Innkeeper client", err)
			return nil, err
		}

		clients = append(clients, ic)
	}

	if len(o.EtcdUrls) > 0 {
		etcdClient, err := etcd.New(etcd.Options{
			Endpoints:  o.EtcdUrls,
			Prefix:     o.EtcdPrefix,
			Timeout:    o.EtcdWaitTimeout,
			Insecure:   o.EtcdInsecure,
			OAuthToken: o.EtcdOAuthToken,
			Username:   o.EtcdUsername,
			Password:   o.EtcdPassword,
		})

		if err != nil {
			return nil, err
		}

		clients = append(clients, etcdClient)
	}

	if o.Kubernetes {
		kubernetesClient, err := kubernetes.New(kubernetes.Options{
			KubernetesIngressV1:               o.KubernetesIngressV1,
			AllowedExternalNames:              o.KubernetesAllowedExternalNames,
			BackendNameTracingTag:             o.OpenTracingBackendNameTag,
			DefaultFiltersDir:                 o.DefaultFiltersDir,
			KubernetesInCluster:               o.KubernetesInCluster,
			KubernetesURL:                     o.KubernetesURL,
			KubernetesNamespace:               o.KubernetesNamespace,
			KubernetesEnableEastWest:          o.KubernetesEnableEastWest,
			KubernetesEastWestDomain:          o.KubernetesEastWestDomain,
			KubernetesEastWestRangeDomains:    o.KubernetesEastWestRangeDomains,
			KubernetesEastWestRangePredicates: o.KubernetesEastWestRangePredicates,
			HTTPSRedirectCode:                 o.KubernetesHTTPSRedirectCode,
			IngressClass:                      o.KubernetesIngressClass,
			OnlyAllowedExternalNames:          o.KubernetesOnlyAllowedExternalNames,
			OriginMarker:                      o.EnableRouteCreationMetrics,
			PathMode:                          o.KubernetesPathMode,
			ProvideHealthcheck:                o.KubernetesHealthcheck,
			ProvideHTTPSRedirect:              o.KubernetesHTTPSRedirect,
			ReverseSourcePredicate:            o.ReverseSourcePredicate,
			RouteGroupClass:                   o.KubernetesRouteGroupClass,
			WhitelistedHealthCheckCIDR:        o.WhitelistedHealthCheckCIDR,
		})
		if err != nil {
			return nil, err
		}
		clients = append(clients, kubernetesClient)
	}

	return clients, nil
}

func getLogOutput(name string) (io.Writer, error) {
	name = path.Clean(name)

	if name == "/dev/stdout" {
		return os.Stdout, nil
	}

	if name == "/dev/stderr" {
		return os.Stderr, nil
	}

	return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
}

func initLog(o Options) error {
	var (
		logOutput       io.Writer
		accessLogOutput io.Writer
		err             error
	)

	if o.ApplicationLogOutput != "" {
		logOutput, err = getLogOutput(o.ApplicationLogOutput)
		if err != nil {
			return err
		}
	}

	if !o.AccessLogDisabled && o.AccessLogOutput != "" {
		accessLogOutput, err = getLogOutput(o.AccessLogOutput)
		if err != nil {
			return err
		}
	}

	logging.Init(logging.Options{
		ApplicationLogPrefix:        o.ApplicationLogPrefix,
		ApplicationLogOutput:        logOutput,
		ApplicationLogJSONEnabled:   o.ApplicationLogJSONEnabled,
		ApplicationLogJsonFormatter: o.ApplicationLogJsonFormatter,
		AccessLogOutput:             accessLogOutput,
		AccessLogJSONEnabled:        o.AccessLogJSONEnabled,
		AccessLogStripQuery:         o.AccessLogStripQuery,
		AccessLogJsonFormatter:      o.AccessLogJsonFormatter,
	})

	return nil
}

func (o *Options) tlsConfig() (*tls.Config, error) {
	if o.ProxyTLS != nil {
		return o.ProxyTLS, nil
	}

	if o.CertPathTLS == "" && o.KeyPathTLS == "" {
		return nil, nil
	}

	crts := strings.Split(o.CertPathTLS, ",")
	keys := strings.Split(o.KeyPathTLS, ",")

	if len(crts) != len(keys) {
		return nil, fmt.Errorf("number of certificates does not match number of keys")
	}

	config := &tls.Config{
		MinVersion: o.TLSMinVersion,
	}

	for i := 0; i < len(crts); i++ {
		crt, key := crts[i], keys[i]
		keypair, err := tls.LoadX509KeyPair(crt, key)
		if err != nil {
			return nil, fmt.Errorf("failed to load X509 keypair from %s and %s: %w", crt, key, err)
		}
		config.Certificates = append(config.Certificates, keypair)
	}
	return config, nil
}

func listen(o *Options, mtr metrics.Metrics) (net.Listener, error) {
	if o.Address == "" {
		o.Address = ":http"
	}

	if !o.EnableTCPQueue {
		return net.Listen("tcp", o.Address)
	}

	var memoryLimit int
	if o.MaxTCPListenerConcurrency <= 0 {
		// cgroup v1: https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
		// cgroup v2: TODO(sszuecs) has to wait for docker/k8s check path /sys/fs/cgroup/<name>/memory.max
		// Note that in containers this will be the container limit.
		// Runtimes without the file will use defaults defined in `queuelistener` package.
		const memoryLimitFile = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
		memoryLimitBytes, err := os.ReadFile(memoryLimitFile)
		if err != nil {
			log.Errorf("Failed to read memory limits, fallback to defaults: %v", err)
		} else {
			memoryLimitString := strings.TrimSpace(string(memoryLimitBytes))
			memoryLimit, err = strconv.Atoi(memoryLimitString)
			if err != nil {
				log.Errorf("Failed to convert memory limits, fallback to defaults: %v", err)
			}

			// 1GB, temporarily, as a tested magic number until a better mechanism is in place:
			if memoryLimit > 1<<30 {
				memoryLimit = 1 << 30
			}
		}
	}

	qto := o.ReadHeaderTimeoutServer
	if qto <= 0 {
		qto = o.ReadTimeoutServer
	}

	return queuelistener.Listen(queuelistener.Options{
		Network:          "tcp",
		Address:          o.Address,
		MaxConcurrency:   o.MaxTCPListenerConcurrency,
		MaxQueueSize:     o.MaxTCPListenerQueue,
		MemoryLimitBytes: memoryLimit,
		ConnectionBytes:  o.ExpectedBytesPerRequest,
		QueueTimeout:     qto,
		Metrics:          mtr,
	})
}

func listenAndServeQuit(
	proxy http.Handler,
	o *Options,
	sigs chan os.Signal,
	idleConnsCH chan struct{},
	mtr metrics.Metrics,
) error {
	tlsConfig, err := o.tlsConfig()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              o.Address,
		TLSConfig:         tlsConfig,
		Handler:           proxy,
		ReadTimeout:       o.ReadTimeoutServer,
		ReadHeaderTimeout: o.ReadHeaderTimeoutServer,
		WriteTimeout:      o.WriteTimeoutServer,
		IdleTimeout:       o.IdleTimeoutServer,
		MaxHeaderBytes:    o.MaxHeaderBytes,
		ErrorLog:          newServerErrorLog(),
	}

	if o.EnableConnMetricsServer {
		m := metrics.Default
		srv.ConnState = func(conn net.Conn, state http.ConnState) {
			m.IncCounter(fmt.Sprintf("lb-conn-%s", state))
		}
	}

	// making idleConnsCH and sigs optional parameters is required to be able to tear down a server
	// from the tests
	if idleConnsCH == nil {
		idleConnsCH = make(chan struct{})
	}

	if sigs == nil {
		sigs = make(chan os.Signal, 1)
	}

	go func() {
		signal.Notify(sigs, syscall.SIGTERM)

		<-sigs

		log.Infof("Got shutdown signal, wait %v for health check", o.WaitForHealthcheckInterval)
		time.Sleep(o.WaitForHealthcheckInterval)

		log.Info("Start shutdown")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Errorf("Failed to graceful shutdown: %v", err)
		}
		close(idleConnsCH)
	}()

	log.Infof("proxy listener on %v", o.Address)

	if srv.TLSConfig != nil {
		if err := srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			log.Errorf("ListenAndServeTLS failed: %v", err)
			return err
		}
	} else {
		log.Infof("TLS settings not found, defaulting to HTTP")

		l, err := listen(o, mtr)
		if err != nil {
			return err
		}

		if err := srv.Serve(l); err != http.ErrServerClosed {
			log.Errorf("Serve failed: %v", err)
			return err
		}
	}

	<-idleConnsCH
	log.Infof("done.")
	return nil
}

func listenAndServe(proxy http.Handler, o *Options) error {
	return listenAndServeQuit(proxy, o, nil, nil, nil)
}

func run(o Options, sig chan os.Signal, idleConnsCH chan struct{}) error {
	// init log
	err := initLog(o)
	if err != nil {
		return err
	}

	if o.EnablePrometheusMetrics {
		o.MetricsFlavours = append(o.MetricsFlavours, "prometheus")
	}

	metricsKind := metrics.UnkownKind
	for _, s := range o.MetricsFlavours {
		switch s {
		case "codahale":
			metricsKind |= metrics.CodaHaleKind
		case "prometheus":
			metricsKind |= metrics.PrometheusKind
		}
	}

	// set default if unset
	if metricsKind == metrics.UnkownKind {
		metricsKind = metrics.CodaHaleKind
	}

	log.Infof("Expose metrics in %s format", metricsKind)
	mtrOpts := metrics.Options{
		Format:                             metricsKind,
		Prefix:                             o.MetricsPrefix,
		EnableDebugGcMetrics:               o.EnableDebugGcMetrics,
		EnableRuntimeMetrics:               o.EnableRuntimeMetrics,
		EnableServeRouteMetrics:            o.EnableServeRouteMetrics,
		EnableServeRouteCounter:            o.EnableServeRouteCounter,
		EnableServeHostMetrics:             o.EnableServeHostMetrics,
		EnableServeHostCounter:             o.EnableServeHostCounter,
		EnableServeMethodMetric:            o.EnableServeMethodMetric,
		EnableServeStatusCodeMetric:        o.EnableServeStatusCodeMetric,
		EnableBackendHostMetrics:           o.EnableBackendHostMetrics,
		EnableProfile:                      o.EnableProfile,
		BlockProfileRate:                   o.BlockProfileRate,
		MutexProfileFraction:               o.MutexProfileFraction,
		MemProfileRate:                     o.MemProfileRate,
		EnableAllFiltersMetrics:            o.EnableAllFiltersMetrics,
		EnableCombinedResponseMetrics:      o.EnableCombinedResponseMetrics,
		EnableRouteResponseMetrics:         o.EnableRouteResponseMetrics,
		EnableRouteBackendErrorsCounters:   o.EnableRouteBackendErrorsCounters,
		EnableRouteStreamingErrorsCounters: o.EnableRouteStreamingErrorsCounters,
		EnableRouteBackendMetrics:          o.EnableRouteBackendMetrics,
		UseExpDecaySample:                  o.MetricsUseExpDecaySample,
		HistogramBuckets:                   o.HistogramMetricBuckets,
		DisableCompatibilityDefaults:       o.DisableMetricsCompatibilityDefaults,
		PrometheusRegistry:                 o.PrometheusRegistry,
	}

	mtr := o.MetricsBackend
	if mtr == nil {
		mtr = metrics.NewMetrics(mtrOpts)
	}
	metrics.Default = mtr

	// *DEPRECATED* client tracking parameter
	if o.ApiUsageMonitoringDefaultClientTrackingPattern != "" {
		log.Warn(`"ApiUsageMonitoringDefaultClientTrackingPattern" option is deprecated`)
	}

	// *DEPRECATED* create authentication for Innkeeper
	inkeeperAuth := innkeeper.CreateInnkeeperAuthentication(innkeeper.AuthOptions{
		InnkeeperAuthToken:  o.InnkeeperAuthToken,
		OAuthCredentialsDir: o.OAuthCredentialsDir,
		OAuthUrl:            o.OAuthUrl,
		OAuthScope:          o.OAuthScope})

	var lbInstance *loadbalancer.LB
	if o.LoadBalancerHealthCheckInterval != 0 {
		lbInstance = loadbalancer.New(o.LoadBalancerHealthCheckInterval)
	}

	if err := o.findAndLoadPlugins(); err != nil {
		return err
	}

	// *DEPRECATED* innkeeper - create data clients
	dataClients, err := createDataClients(o, inkeeperAuth)
	if err != nil {
		return err
	}

	// append custom data clients
	dataClients = append(dataClients, o.CustomDataClients...)

	if len(dataClients) == 0 {
		log.Warning("no route source specified")
	}

	o.PluginDirs = append(o.PluginDirs, o.PluginDir)

	var tracer ot.Tracer
	if len(o.OpenTracing) > 0 {
		tracer, err = tracing.InitTracer(o.OpenTracing)
		if err != nil {
			return err
		}
	} else {
		// always have a tracer available, so filter authors can rely on the
		// existence of a tracer
		tracer, _ = tracing.LoadTracingPlugin(o.PluginDirs, []string{"noop"})
	}

	if o.OAuthTokeninfoURL != "" {
		tio := auth.TokeninfoOptions{
			URL:          o.OAuthTokeninfoURL,
			Timeout:      o.OAuthTokeninfoTimeout,
			MaxIdleConns: o.IdleConnectionsPerHost,
			Tracer:       tracer,
		}

		o.CustomFilters = append(o.CustomFilters,
			auth.NewOAuthTokeninfoAllScopeWithOptions(tio),
			auth.NewOAuthTokeninfoAnyScopeWithOptions(tio),
			auth.NewOAuthTokeninfoAllKVWithOptions(tio),
			auth.NewOAuthTokeninfoAnyKVWithOptions(tio),
		)
	}

	if o.SecretsRegistry == nil {
		o.SecretsRegistry = secrets.NewRegistry()
	}
	defer o.SecretsRegistry.Close()

	sp := secrets.NewSecretPaths(o.CredentialsUpdateInterval)
	defer sp.Close()
	for _, p := range o.CredentialsPaths {
		if err := sp.Add(p); err != nil {
			log.Errorf("Failed to add credentials file: %s: %v", p, err)
		}
	}

	tio := auth.TokenintrospectionOptions{
		Timeout:      o.OAuthTokenintrospectionTimeout,
		MaxIdleConns: o.IdleConnectionsPerHost,
		Tracer:       tracer,
	}

	who := auth.WebhookOptions{
		Timeout:      o.WebhookTimeout,
		MaxIdleConns: o.IdleConnectionsPerHost,
		Tracer:       tracer,
	}

	o.CustomFilters = append(o.CustomFilters,
		logfilter.NewAuditLog(o.MaxAuditBody),
		auth.NewBearerInjector(sp),
		auth.NewJwtValidationWithOptions(tio),
		auth.TokenintrospectionWithOptions(auth.NewOAuthTokenintrospectionAnyClaims, tio),
		auth.TokenintrospectionWithOptions(auth.NewOAuthTokenintrospectionAllClaims, tio),
		auth.TokenintrospectionWithOptions(auth.NewOAuthTokenintrospectionAnyKV, tio),
		auth.TokenintrospectionWithOptions(auth.NewOAuthTokenintrospectionAllKV, tio),
		auth.TokenintrospectionWithOptions(auth.NewSecureOAuthTokenintrospectionAnyClaims, tio),
		auth.TokenintrospectionWithOptions(auth.NewSecureOAuthTokenintrospectionAllClaims, tio),
		auth.TokenintrospectionWithOptions(auth.NewSecureOAuthTokenintrospectionAnyKV, tio),
		auth.TokenintrospectionWithOptions(auth.NewSecureOAuthTokenintrospectionAllKV, tio),
		auth.WebhookWithOptions(who),
		auth.NewOAuthOidcUserInfos(o.OIDCSecretsFile, o.SecretsRegistry),
		auth.NewOAuthOidcAnyClaims(o.OIDCSecretsFile, o.SecretsRegistry),
		auth.NewOAuthOidcAllClaims(o.OIDCSecretsFile, o.SecretsRegistry),
		auth.NewOIDCQueryClaimsFilter(),
		apiusagemonitoring.NewApiUsageMonitoring(
			o.ApiUsageMonitoringEnable,
			o.ApiUsageMonitoringRealmKeys,
			o.ApiUsageMonitoringClientKeys,
			o.ApiUsageMonitoringRealmsTrackingPattern,
		),
	)

	var swarmer ratelimit.Swarmer
	var redisOptions *skpnet.RedisOptions
	if o.EnableSwarm {
		if len(o.SwarmRedisURLs) > 0 {
			log.Infof("Redis based swarm with %d shards", len(o.SwarmRedisURLs))
			redisOptions = &skpnet.RedisOptions{
				Addrs:               o.SwarmRedisURLs,
				Password:            o.SwarmRedisPassword,
				HashAlgorithm:       o.SwarmRedisHashAlgorithm,
				DialTimeout:         o.SwarmRedisDialTimeout,
				ReadTimeout:         o.SwarmRedisReadTimeout,
				WriteTimeout:        o.SwarmRedisWriteTimeout,
				PoolTimeout:         o.SwarmRedisPoolTimeout,
				MinIdleConns:        o.SwarmRedisMinIdleConns,
				MaxIdleConns:        o.SwarmRedisMaxIdleConns,
				ConnMetricsInterval: o.redisConnMetricsInterval,
				Tracer:              tracer,
			}
		} else {
			log.Infof("Start swim based swarm")
			swops := &swarm.Options{
				SwarmPort:        uint16(o.SwarmPort),
				MaxMessageBuffer: o.SwarmMaxMessageBuffer,
				LeaveTimeout:     o.SwarmLeaveTimeout,
				Debug:            log.GetLevel() == log.DebugLevel,
			}

			if o.Kubernetes {
				swops.KubernetesOptions = &swarm.KubernetesOptions{
					KubernetesInCluster:  o.KubernetesInCluster,
					KubernetesAPIBaseURL: o.KubernetesURL,
					Namespace:            o.SwarmKubernetesNamespace,
					LabelSelectorKey:     o.SwarmKubernetesLabelSelectorKey,
					LabelSelectorValue:   o.SwarmKubernetesLabelSelectorValue,
				}
			}

			if o.SwarmStaticSelf != "" {
				self, err := swarm.NewStaticNodeInfo(o.SwarmStaticSelf, o.SwarmStaticSelf)
				if err != nil {
					log.Fatalf("Failed to get static NodeInfo: %v", err)
				}
				other := []*swarm.NodeInfo{self}

				for _, addr := range strings.Split(o.SwarmStaticOther, ",") {
					ni, err := swarm.NewStaticNodeInfo(addr, addr)
					if err != nil {
						log.Fatalf("Failed to get static NodeInfo: %v", err)
					}
					other = append(other, ni)
				}

				swops.StaticSwarm = swarm.NewStaticSwarm(self, other)
			}

			theSwarm, err := swarm.NewSwarm(swops)
			if err != nil {
				log.Errorf("failed to init swarm with options %+v: %v", swops, err)
			}
			defer theSwarm.Leave()
			swarmer = theSwarm
		}
	}

	var ratelimitRegistry *ratelimit.Registry
	if o.EnableRatelimiters || len(o.RatelimitSettings) > 0 {
		log.Infof("enabled ratelimiters %v: %v", o.EnableRatelimiters, o.RatelimitSettings)
		ratelimitRegistry = ratelimit.NewSwarmRegistry(swarmer, redisOptions, o.RatelimitSettings...)
		defer ratelimitRegistry.Close()

		if hook := o.SwarmRegistry; hook != nil {
			hook(ratelimitRegistry)
		}

		if o.ClusterRatelimitMaxGroupShards < 1 {
			log.Warn("ClusterRatelimitMaxGroupShards must be positive, reset to 1")
			o.ClusterRatelimitMaxGroupShards = 1
		}

		provider := ratelimitfilters.NewRatelimitProvider(ratelimitRegistry)
		o.CustomFilters = append(o.CustomFilters,
			ratelimitfilters.NewClientRatelimit(provider),
			ratelimitfilters.NewLocalRatelimit(provider),
			ratelimitfilters.NewRatelimit(provider),
			ratelimitfilters.NewShardedClusterRateLimit(provider, o.ClusterRatelimitMaxGroupShards),
			ratelimitfilters.NewClusterClientRateLimit(provider),
			ratelimitfilters.NewDisableRatelimit(provider),
			ratelimitfilters.NewBackendRatelimit(),
		)
	}

	if o.TLSMinVersion == 0 {
		o.TLSMinVersion = tls.VersionTLS12
	}

	oauthConfig := &auth.OAuthConfig{}
	if o.EnableOAuth2GrantFlow /* explicitly enable grant flow */ {
		grantSecrets := secrets.NewSecretPaths(o.CredentialsUpdateInterval)
		defer grantSecrets.Close()

		oauthConfig.AuthURL = o.OAuth2AuthURL
		oauthConfig.TokenURL = o.OAuth2TokenURL
		oauthConfig.RevokeTokenURL = o.OAuth2RevokeTokenURL
		oauthConfig.TokeninfoURL = o.OAuthTokeninfoURL
		oauthConfig.SecretFile = o.OAuth2SecretFile
		oauthConfig.ClientID = o.OAuth2ClientID
		oauthConfig.ClientSecret = o.OAuth2ClientSecret
		oauthConfig.ClientIDFile = o.OAuth2ClientIDFile
		oauthConfig.ClientSecretFile = o.OAuth2ClientSecretFile
		oauthConfig.CallbackPath = o.OAuth2CallbackPath
		oauthConfig.AuthURLParameters = o.OAuth2AuthURLParameters
		oauthConfig.SecretsProvider = grantSecrets
		oauthConfig.Secrets = o.SecretsRegistry
		oauthConfig.AccessTokenHeaderName = o.OAuth2AccessTokenHeaderName
		oauthConfig.TokeninfoSubjectKey = o.OAuth2TokeninfoSubjectKey
		oauthConfig.TokenCookieName = o.OAuth2TokenCookieName
		oauthConfig.ConnectionTimeout = o.OAuthTokeninfoTimeout
		oauthConfig.MaxIdleConnectionsPerHost = o.IdleConnectionsPerHost
		oauthConfig.Tracer = tracer

		if err := oauthConfig.Init(); err != nil {
			log.Errorf("Failed to initialize oauth grant filter: %v.", err)
			return err
		}

		o.CustomFilters = append(o.CustomFilters,
			oauthConfig.NewGrant(),
			oauthConfig.NewGrantCallback(),
			oauthConfig.NewGrantClaimsQuery(),
			oauthConfig.NewGrantLogout(),
		)
	}

	if len(o.CompressEncodings) > 0 {
		compress, err := builtin.NewCompressWithOptions(builtin.CompressOptions{Encodings: o.CompressEncodings})
		if err != nil {
			log.Errorf("Failed to create compress filter: %v.", err)
			return err
		}
		o.CustomFilters = append(o.CustomFilters, compress)
	}

	// create a filter registry with the available filter specs registered,
	// and register the custom filters
	registry := builtin.MakeRegistry()
	for _, f := range o.CustomFilters {
		registry.Register(f)
	}

	// create routing
	// create the proxy instance
	var mo routing.MatchingOptions
	if o.IgnoreTrailingSlash {
		mo = routing.IgnoreTrailingSlash
	}

	// ensure a non-zero poll timeout
	if o.SourcePollTimeout <= 0 {
		o.SourcePollTimeout = defaultSourcePollTimeout
	}

	// check for dev mode, and set update buffer of the routes
	updateBuffer := defaultRoutingUpdateBuffer
	if o.DevMode {
		updateBuffer = 0
	}

	// include bundled custom predicates
	o.CustomPredicates = append(o.CustomPredicates,
		source.New(),
		source.NewFromLast(),
		source.NewClientIP(),
		interval.NewBetween(),
		interval.NewBefore(),
		interval.NewAfter(),
		cron.New(),
		cookie.New(),
		query.New(),
		traffic.New(),
		primitive.NewTrue(),
		primitive.NewFalse(),
		primitive.NewShutdown(),
		pauth.NewJWTPayloadAllKV(),
		pauth.NewJWTPayloadAnyKV(),
		pauth.NewJWTPayloadAllKVRegexp(),
		pauth.NewJWTPayloadAnyKVRegexp(),
		methods.New(),
		tee.New(),
		forwarded.NewForwardedHost(),
		forwarded.NewForwardedProto(),
		host.NewAny(),
	)

	// provide default value for wrapper if not defined
	if o.CustomHttpHandlerWrap == nil {
		o.CustomHttpHandlerWrap = func(original http.Handler) http.Handler {
			return original
		}
	}

	schedulerRegistry := scheduler.RegistryWith(scheduler.Options{
		Metrics:                mtr,
		EnableRouteLIFOMetrics: o.EnableRouteLIFOMetrics,
	})
	defer schedulerRegistry.Close()

	// create a routing engine
	ro := routing.Options{
		FilterRegistry:  registry,
		MatchingOptions: mo,
		PollTimeout:     o.SourcePollTimeout,
		DataClients:     dataClients,
		Predicates:      o.CustomPredicates,
		UpdateBuffer:    updateBuffer,
		SuppressLogs:    o.SuppressRouteUpdateLogs,
		PostProcessors: []routing.PostProcessor{
			loadbalancer.HealthcheckPostProcessor{LB: lbInstance},
			loadbalancer.NewAlgorithmProvider(),
			schedulerRegistry,
			builtin.NewRouteCreationMetrics(mtr),
			fadein.NewPostProcessor(),
		},
		SignalFirstLoad: o.WaitFirstRouteLoad,
	}

	if o.DefaultFilters != nil {
		ro.PreProcessors = append(ro.PreProcessors, o.DefaultFilters)
	}

	if o.CloneRoute != nil {
		ro.PreProcessors = append(ro.PreProcessors, o.CloneRoute)
	}

	if o.EditRoute != nil {
		ro.PreProcessors = append(ro.PreProcessors, o.EditRoute)
	}

	ro.PreProcessors = append(ro.PreProcessors, schedulerRegistry.PreProcessor())

	if o.EnableOAuth2GrantFlow /* explicitly enable grant flow when callback route was not disabled */ {
		ro.PreProcessors = append(ro.PreProcessors, oauthConfig.NewGrantPreprocessor())
	}

	routing := routing.New(ro)
	defer routing.Close()

	proxyFlags := proxy.Flags(o.ProxyOptions) | o.ProxyFlags
	proxyParams := proxy.Params{
		Routing:                    routing,
		Flags:                      proxyFlags,
		PriorityRoutes:             o.PriorityRoutes,
		IdleConnectionsPerHost:     o.IdleConnectionsPerHost,
		CloseIdleConnsPeriod:       o.CloseIdleConnsPeriod,
		FlushInterval:              o.BackendFlushInterval,
		ExperimentalUpgrade:        o.ExperimentalUpgrade,
		ExperimentalUpgradeAudit:   o.ExperimentalUpgradeAudit,
		MaxLoopbacks:               o.MaxLoopbacks,
		DefaultHTTPStatus:          o.DefaultHTTPStatus,
		LoadBalancer:               lbInstance,
		Timeout:                    o.TimeoutBackend,
		ResponseHeaderTimeout:      o.ResponseHeaderTimeoutBackend,
		ExpectContinueTimeout:      o.ExpectContinueTimeoutBackend,
		KeepAlive:                  o.KeepAliveBackend,
		DualStack:                  o.DualStackBackend,
		TLSHandshakeTimeout:        o.TLSHandshakeTimeoutBackend,
		MaxIdleConns:               o.MaxIdleConnsBackend,
		DisableHTTPKeepalives:      o.DisableHTTPKeepalives,
		AccessLogDisabled:          o.AccessLogDisabled,
		ClientTLS:                  o.ClientTLS,
		CustomHttpRoundTripperWrap: o.CustomHttpRoundTripperWrap,
		RateLimiters:               ratelimitRegistry,
	}

	if o.EnableBreakers || len(o.BreakerSettings) > 0 {
		proxyParams.CircuitBreakers = circuit.NewRegistry(o.BreakerSettings...)
	}

	if o.DebugListener != "" {
		do := proxyParams
		do.Flags |= proxy.Debug
		dbg := proxy.WithParams(do)
		log.Infof("debug listener on %v", o.DebugListener)
		go func() { http.ListenAndServe(o.DebugListener, dbg) }()
	}

	// init support endpoints
	supportListener := o.SupportListener

	// Backward compatibility
	if supportListener == "" {
		supportListener = o.MetricsListener
	}

	if supportListener != "" {
		mux := http.NewServeMux()
		mux.Handle("/routes", routing)
		mux.Handle("/routes/", routing)

		metricsHandler := metrics.NewHandler(mtrOpts, mtr)
		mux.Handle("/metrics", metricsHandler)
		mux.Handle("/metrics/", metricsHandler)
		mux.Handle("/debug/pprof", metricsHandler)
		mux.Handle("/debug/pprof/", metricsHandler)

		log.Infof("support listener on %s", supportListener)
		go func() {
			if err := http.ListenAndServe(supportListener, mux); err != nil {
				log.Errorf("Failed to start supportListener on %s: %v", supportListener, err)
			}
		}()
	} else {
		log.Infoln("Metrics are disabled")
	}

	proxyParams.OpenTracing = &proxy.OpenTracingParams{
		Tracer:             tracer,
		InitialSpan:        o.OpenTracingInitialSpan,
		ExcludeTags:        o.OpenTracingExcludedProxyTags,
		DisableFilterSpans: o.OpenTracingDisableFilterSpans,
		LogFilterEvents:    o.OpenTracingLogFilterLifecycleEvents,
		LogStreamEvents:    o.OpenTracingLogStreamEvents,
	}

	// create the proxy
	proxy := proxy.WithParams(proxyParams)
	defer proxy.Close()

	for _, startupCheckURL := range o.StatusChecks {
		for {
			/* #nosec */
			resp, err := http.Get(startupCheckURL)
			if err != nil {
				log.Infof("%s unhealthy", startupCheckURL)
				time.Sleep(1 * time.Second)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Infof("%s healthy", startupCheckURL)
				break
			}
			log.Infof("%s unhealthy", startupCheckURL)
			time.Sleep(1 * time.Second)
		}
	}

	// wait for the first route configuration to be loaded if enabled:
	<-routing.FirstLoad()

	return listenAndServeQuit(o.CustomHttpHandlerWrap(proxy), &o, sig, idleConnsCH, mtr)
}

// Run skipper.
func Run(o Options) error {
	return run(o, nil, nil)
}
