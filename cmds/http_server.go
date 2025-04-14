package cmds

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Depado/ginprom"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/aurowora/compress"
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
	"github.com/rm-hull/gps-routes-api/services/osdatahub"
)

func NewHttpServer(port int) {

	// Connect to Postgres
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbConfig := db.ConfigFromEnv()
	pool, err := db.NewDBPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	engine := gin.New()

	prometheus := ginprom.New(
		ginprom.Engine(engine),
		ginprom.Namespace("gps_routes"),
		ginprom.Subsystem("api"),
	)

	rlStore := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 5,
	})
	rateLimiter := ratelimit.RateLimiter(rlStore, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.JSON(429, gin.H{
				"error":   "Too many requests",
				"message": "Try again in " + time.Until(info.ResetTime).String(),
			})
		},
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	})

	engine.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz"),
		gin.Recovery(),
		cors.Default(),
		middlewares.ErrorHandler(),
		compress.Compress(),
		limits.RequestSizeLimiter(10*1024),
		cachecontrol.New(cachecontrol.CacheAssetsForeverPreset),
		prometheus.Instrument(),
		rateLimiter,
	)

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()
	err = healthcheck.New(engine, hc_config.DefaultConfig(), []checks.Check{
		checks.SqlCheck{Sql: db},
	})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	namesApi := osdatahub.NewNamesApi(prometheus, "https://api.os.uk/search/names/v1", os.Getenv("OS_NAMES_API_KEY"))
	pg := repositories.NewPostgresRouteRepository(pool, dbConfig.Schema)
	repo := repositories.NewCachedRepository(prometheus, pg)
	service := services.NewRoutesService(repo, namesApi)

	router := routes.NewRouterWithGinEngine(engine, routes.ApiHandleFunctions{
		RoutesAPI: routes.RoutesAPI{
			Service: service,
		},
	})

	addr := fmt.Sprintf(":%d", port)
	log.Println("HTTP API Server started on port", addr)
	log.Fatal(router.Run(addr))
}
