package cmds

import (
	"context"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/ikeikeikeike/go-sitemap-generator/stm"
	"github.com/rm-hull/gps-routes-api/db"
	openapi "github.com/rm-hull/gps-routes-api/go"
	"github.com/rm-hull/gps-routes-api/repositories"
	"github.com/schollz/progressbar/v3"
)

func GenerateSitemap(host string) {
	dbConfig := db.ConfigFromEnv()
	ctx := context.Background()
	pool, err := db.NewDBPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	pg := repositories.NewPostgresRouteRepository(pool, dbConfig.Schema)
	routes, err := pg.SearchHits(ctx, &openapi.SearchRequest{Limit: 1_000_000})
	if err != nil {
		log.Fatalf("Failed to get routes: %v", err)
	}
	sm := stm.NewSitemap()
	sm.SetDefaultHost(host)
	sm.SetPretty(true)
	sm.SetCompress(true)
	sm.Create()

	now := time.Now().Format("2006-01-02")
	bar := progressbar.Default(int64(len(*routes)))
	for _, route := range *routes {
		url := stm.URL{
			"loc":     url.QueryEscape(route.Ref),
			"lastmod": now,
		}

		if route.HeadlineImageUrl != nil {
			url["image"] = []stm.URL{{
				"loc": *route.HeadlineImageUrl,
			}}
		}
		sm.Add(url)
		bar.Add(1)
	}

	// Write sitemap to file
	f, err := os.Create("sitemap.xml")
	if err != nil {
		log.Fatalf("failed to create sitemap file: %v", err)
	}
	defer f.Close()

	if _, err := f.Write(sm.XMLContent()); err != nil {
		log.Fatalf("failed to write sitemap: %v", err)
	}

	log.Printf("Sitemap generated")
}
