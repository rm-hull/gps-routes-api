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
	"os"

	"github.com/earthboundkid/versioninfo/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	cmds "github.com/rm-hull/gps-routes-api/cmd"
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
			fmt.Printf("TODO: Importing data from: %s\n", args[0])
		},
	}

	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			cmds.NewHttpServer()
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
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
