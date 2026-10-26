// Package app assembles the business API. It has no Agent or vector-service dependency.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/auth"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/cart"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/catalog"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/files"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/httpapi"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/knowledge"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/order"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/platform/configcenter"
	mysqlstore "github.com/LYP-leo/xzxg-shop/commerce/internal/platform/mysql"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/promotion"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/review"
)

type Application struct {
	Address string
	Handler http.Handler
	store   *mysqlstore.MySQLStore
}

func (a *Application) Close() error { return a.store.Close() }

func New(ctx context.Context, logger *slog.Logger) (*Application, error) {
	production := strings.EqualFold(env("APP_ENV", "development"), "production")
	dsn := env("MYSQL_DSN", "root:root@tcp(127.0.0.1:3307)/xzxg_commerce?parseTime=true&loc=Local")
	if production && (strings.HasPrefix(dsn, "root:root@") || os.Getenv("MYSQL_DSN") == "") {
		return nil, errors.New("production requires an explicit MYSQL_DSN without default credentials")
	}
	repo, err := mysqlstore.OpenMySQL(ctx, dsn)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = repo.Close()
		}
	}()
	if envBool("RUN_MIGRATIONS", !production) {
		if err := repo.Migrate(ctx); err != nil {
			return nil, err
		}
	}
	if envBool("SEED_DEMO_DATA", !production) {
		if production {
			return nil, errors.New("demo data is disabled in production")
		}
		if err := repo.Seed(ctx); err != nil {
			return nil, err
		}
	}
	defaults := configcenter.DefaultConfigs()
	for i := range defaults {
		key := "COMMERCE_" + strings.ToUpper(strings.ReplaceAll(defaults[i].ConfigKey, ".", "_"))
		if value, exists := os.LookupEnv(key); exists {
			defaults[i].ConfigValue = value
		}
	}
	var center configcenter.Center = configcenter.NewMemoryCenter(defaults)
	if addr := os.Getenv("NACOS_ADDR"); addr != "" {
		nacos := configcenter.NewNacosCenter(addr, os.Getenv("NACOS_NAMESPACE"), "XZXG_COMMERCE", "xzxg-commerce-app-config.json", defaults)
		if err := nacos.Seed(ctx); err != nil {
			return nil, err
		}
		center = nacos
	}
	if production {
		values := center.GetMap(ctx)
		if strings.Contains(values["http.cors.allowed_origins"], "*") {
			return nil, errors.New("production requires explicit CORS origins")
		}
		if values["minio.access_key"] == "minioadmin" || values["minio.secret_key"] == "minioadmin" {
			return nil, errors.New("production requires non-default MinIO credentials")
		}
	}
	services := httpapi.Services{
		Auth: auth.New(repo), Catalog: catalog.New(repo), Cart: cart.New(repo),
		Order: order.New(repo), Promotion: promotion.New(repo), Review: review.New(repo),
		Knowledge: knowledge.New(repo), Files: files.New(repo),
	}
	ok = true
	return &Application{Address: env("API_ADDR", ":8081"), Handler: httpapi.NewServer(services, center, logger).Routes(), store: repo}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}
