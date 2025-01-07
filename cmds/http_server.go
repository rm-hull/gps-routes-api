package cmds

import (
	"context"
	"log"
	"time"

	"github.com/aurowora/compress"
	"github.com/earthboundkid/versioninfo/v2"
	"github.com/gin-contrib/cors"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/stdlib"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"

	"github.com/rm-hull/gps-routes-api/db"
	"github.com/rm-hull/gps-routes-api/middlewares"
	"github.com/rm-hull/gps-routes-api/repositories"
	"github.com/rm-hull/gps-routes-api/routes"
	"github.com/rm-hull/gps-routes-api/services"
)

func NewHttpServer() {

	// Connect to Postgres
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := db.ConfigFromEnv()
	pool, err := db.NewDBPool(ctx, config)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	repo := repositories.NewCachedRepository(repositories.NewPostgresRouteRepository(pool, config.Schema))
	service := services.NewRoutesService(repo)

	log.Printf("Server started, version: %s", versioninfo.Short())

	engine := gin.New()
	engine.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz"),
		gin.Recovery(),
		cors.Default(),
		middlewares.ErrorHandler(),
		compress.Compress(),
		limits.RequestSizeLimiter(10*1024),
		cachecontrol.New(cachecontrol.CacheAssetsForeverPreset),
	)

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()
	err = healthcheck.New(engine, hc_config.DefaultConfig(), []checks.Check{
		checks.SqlCheck{Sql: db},
	})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	router := routes.NewRouterWithGinEngine(engine, routes.ApiHandleFunctions{
		RoutesAPI: routes.RoutesAPI{
			Service: service,
		},
	})

	log.Fatal(router.Run(":8080"))
}
