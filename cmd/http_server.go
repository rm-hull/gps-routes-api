package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aurowora/compress"
	"github.com/earthboundkid/versioninfo/v2"
	"github.com/gin-contrib/cors"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	"github.com/tavsec/gin-healthcheck/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rm-hull/gps-routes-api/middlewares"
	"github.com/rm-hull/gps-routes-api/repositories"
	"github.com/rm-hull/gps-routes-api/routes"
	"github.com/rm-hull/gps-routes-api/services"
)

func NewHttpServer() {

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("Environment variable MONGO_URI is not set")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	repo := repositories.NewMongoRouteRepository(client, "gps-routes", "routes")
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
	)

	err = healthcheck.New(engine, config.DefaultConfig(), []checks.Check{
		checks.NewMongoCheck(10, client),
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
