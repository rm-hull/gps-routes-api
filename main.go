/*
 * GPS Routes API
 *
 * API to retrieve and search GPS Walking Routes
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/earthboundkid/versioninfo/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/rm-hull/gps-routes-api/cmds"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var rootCmd = &cobra.Command{
		Use:  "gps-routes",
		Long: `HTTP server, DB migration and data import/export`,
	}

	var importCmd = &cobra.Command{
		Use:   "import [path]",
		Short: "Import JSON data from specified path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmds.ImportData(args[0], math.MaxInt64)
		},
	}

	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			cmds.NewHttpServer()
		},
	}

	var pingDbCmd = &cobra.Command{
		Use:   "ping",
		Short: "Ping Postgres database",
		Run: func(cmd *cobra.Command, args []string) {
			cmds.PingDatabase()
		},
	}

	var migrationCmd = &cobra.Command{
		Use:   "migration [up|down] <migrations_path>",
		Short: "Run DB migration",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cmds.RunMigration(args[0], args[1])
		},
	}

	var sitemapCmd = &cobra.Command{
		Use:   "sitemap [base_host_url]",
		Short: "Generate sitemap.xml",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmds.GenerateSitemap(args[0])
		},
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(versioninfo.Short())
		},
	}

	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(pingDbCmd)
	rootCmd.AddCommand(migrationCmd)
	rootCmd.AddCommand(sitemapCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
