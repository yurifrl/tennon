// Command tennon is the Tennon server: controller + management API + MCP proxy
// + embedded web UI, in one self-contained binary. This file is the single
// composition root — the only place that knows concrete implementations. It
// wires them with Uber FX and owns process lifecycle.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/yurifrl/tennon/api"
	"github.com/yurifrl/tennon/auth"
	"github.com/yurifrl/tennon/controller"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/proxy"
	"github.com/yurifrl/tennon/runtime"
	"github.com/yurifrl/tennon/secret"
	"github.com/yurifrl/tennon/store"
	"github.com/yurifrl/tennon/web"
)

// config holds the server's startup flags.
type config struct {
	storeBackend  string
	storePath     string
	runtime       string
	apiAddr       string
	proxyAddr     string
	namespace     string
	oidcIssuer    string
	oidcClientID  string
	oidcUserClaim string
	oidcGroupClaim string
	oidcProvider  string
	authz         string
}

func parseFlags() config {
	var c config
	flag.StringVar(&c.storeBackend, "store", "sqlite", "store backend: memory|sqlite|postgres|kubernetes")
	flag.StringVar(&c.storePath, "store-path", "tennon.db", "sqlite file path")
	flag.StringVar(&c.runtime, "runtime", "auto", "runtime backend: auto|remote|docker|kubernetes")
	flag.StringVar(&c.apiAddr, "api-addr", ":8080", "API + web UI listen address")
	flag.StringVar(&c.proxyAddr, "proxy-addr", ":8081", "MCP proxy listen address")
	flag.StringVar(&c.namespace, "namespace", "tennon", "default object namespace")
	flag.StringVar(&c.oidcIssuer, "oidc-issuer", "", "OIDC issuer URL (empty = anonymous verifier)")
	flag.StringVar(&c.oidcClientID, "oidc-client-id", "", "OIDC client ID")
	flag.StringVar(&c.oidcUserClaim, "oidc-username-claim", "email", "OIDC claim for username")
	flag.StringVar(&c.oidcGroupClaim, "oidc-groups-claim", "groups", "OIDC claim for groups")
	flag.StringVar(&c.oidcProvider, "oidc-provider", "", "OIDC provider name (e.g., google, okta)")
	flag.StringVar(&c.authz, "authz", "allow-all", "authorization mode: allow-all|policy")
	flag.Parse()
	return c
}

func main() {
	cfg := parseFlags()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	app := fx.New(
		fx.Supply(cfg, logger),
		// Infrastructure — the only place concrete backends are chosen.
		fx.Provide(func(c config) (domain.Store, error) {
			return store.NewFromConfig(store.Config{Backend: c.storeBackend, Path: c.storePath})
		}),
		fx.Provide(func(c config) (domain.Runtime, error) {
			return runtime.NewFromConfig(runtime.Config{Backend: c.runtime})
		}),
		fx.Provide(fx.Annotate(
			func(s domain.Store, c config) *secret.Renderer { return secret.NewRenderer(s, c.namespace) },
			fx.As(new(domain.SecretRenderer)),
		)),
		fx.Provide(fx.Annotate(
			func(c config, s domain.Store, logger *slog.Logger) (domain.Authorizer, error) {
				switch c.authz {
				case "policy":
					return auth.NewPolicyAuthorizer(context.Background(), s, logger)
				case "allow-all":
					return auth.NewAllowAllAuthorizer(), nil
				default:
					return nil, fmt.Errorf("unknown authz mode: %s", c.authz)
				}
			},
			fx.As(new(domain.Authorizer)),
		)),
		fx.Provide(fx.Annotate(
			func(c config, s domain.Store) (domain.IdentityVerifier, error) {
				return auth.NewVerifierFromConfig(context.Background(), auth.OIDCConfig{
					IssuerURL:     c.oidcIssuer,
					ClientID:      c.oidcClientID,
					UsernameClaim: c.oidcUserClaim,
					GroupsClaim:   c.oidcGroupClaim,
					ProviderName:  c.oidcProvider,
				}, s)
			},
			fx.As(new(domain.IdentityVerifier)),
		)),
		// Delivery — the proxy also satisfies domain.Router for the controller and
		// domain.ExecutionCompleter for the API (approval resume).
		fx.Provide(proxy.New),
		fx.Provide(func(p *proxy.Proxy) domain.Router { return p }),
		fx.Provide(func(p *proxy.Proxy) domain.ExecutionCompleter { return p }),
		fx.Provide(func(p *proxy.Proxy) domain.ToolProvider { return p }),
		// Domain services consumed by the API (approval resume, token mint/revoke, catalog).
		fx.Provide(func(s domain.Store) *domain.TokenService { return domain.NewTokenService(s) }),
		fx.Provide(func(tp domain.ToolProvider) *domain.ToolCatalog { return domain.NewToolCatalog(tp) }),
		fx.Provide(api.New),
		fx.Provide(controller.NewManager),
		// Start everything.
		fx.Invoke(registerLifecycle),

		fx.NopLogger,
	)
	if err := app.Err(); err != nil {
		logger.Error("failed to build application", "err", err)
		os.Exit(1)
	}
	app.Run()
}

// registerLifecycle starts the API/web server, the proxy server, and the
// controller in dependency order, and drains them on shutdown.
func registerLifecycle(
	lc fx.Lifecycle,
	cfg config,
	logger *slog.Logger,
	apiSrv *api.Server,
	proxyImpl *proxy.Proxy,
	mgr *controller.Manager,
) {
	// API + web UI on the API address.
	apiMux := http.NewServeMux()
	apiHandler := apiSrv.Handler()
	apiMux.Handle("/api/", apiHandler)
	apiMux.Handle("/healthz", apiHandler)
	apiMux.Handle("/", web.Handler())
	apiHTTP := &http.Server{Addr: cfg.apiAddr, Handler: apiMux, ReadHeaderTimeout: 10 * time.Second}
	proxyHTTP := &http.Server{Addr: cfg.proxyAddr, Handler: proxyImpl.Handler(), ReadHeaderTimeout: 10 * time.Second}

	ctrlCtx, cancelCtrl := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go serve(logger, "api", apiHTTP)
			go serve(logger, "proxy", proxyHTTP)
			go func() {
				if err := mgr.Run(ctrlCtx); err != nil && ctrlCtx.Err() == nil {
					logger.Error("controller stopped", "err", err)
				}
			}()
			logger.Info("tennon started", "api", cfg.apiAddr, "proxy", cfg.proxyAddr, "store", cfg.storeBackend)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancelCtrl()
			// Drain the proxy before the API so in-flight tool calls finish.
			_ = proxyHTTP.Shutdown(ctx)
			_ = apiHTTP.Shutdown(ctx)
			return nil
		},
	})
}

func serve(logger *slog.Logger, name string, srv *http.Server) {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "server", name, "err", err)
	}
}
