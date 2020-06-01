package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kubermatic-labs/aquayman/pkg/config"
	"github.com/kubermatic-labs/aquayman/pkg/export"
	"github.com/kubermatic-labs/aquayman/pkg/quay"
	"github.com/kubermatic-labs/aquayman/pkg/sync"
)

// These variables are set by goreleaser during build time.
var (
	version = "dev"
	date    = "unknown"
)

func main() {
	ctx := context.Background()

	var (
		configFile         = ""
		showVersion        = false
		confirm            = false
		validate           = false
		exportMode         = false
		createRepositories = false
		deleteRepositories = false
	)

	flag.StringVar(&configFile, "config", configFile, "path to the config.yaml")
	flag.BoolVar(&showVersion, "version", showVersion, "show the Aquayman version and exit")
	flag.BoolVar(&confirm, "confirm", confirm, "must be set to actually perform any changes on quay.io")
	flag.BoolVar(&validate, "validate", validate, "validate the given configuration and then exit")
	flag.BoolVar(&exportMode, "export", exportMode, "export quay.io state and update the config file (-config flag)")
	flag.BoolVar(&createRepositories, "create-repos", createRepositories, "create repositories listed in the config file but not existing on quay.io yet")
	flag.BoolVar(&deleteRepositories, "delete-repos", deleteRepositories, "delete repositories on quay.io that are not listed in the config file")
	flag.Parse()

	if showVersion {
		fmt.Printf("Aquayman %s (built at %s)\n", version, date)
		return
	}

	if configFile == "" {
		log.Print("⚠ No configuration (-config) specified.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		log.Fatalf("⚠ Failed to load config %q: %v.", configFile, err)
	}

	if validate {
		log.Println("✓ Configuration is valid.")
		return
	}

	client, err := quay.NewClient(getToken(), 30*time.Second, !confirm)
	if err != nil {
		log.Fatalf("⚠ Failed to create quay.io API client: %v.", err)
	}

	if exportMode {
		log.Printf("► Exporting organization %s…", cfg.Organization)

		newConfig, err := export.ExportConfiguration(ctx, cfg.Organization, client)
		if err != nil {
			log.Fatalf("⚠ Failed to export: %v.", err)
		}

		if err := config.SaveToFile(newConfig, configFile); err != nil {
			log.Fatalf("⚠ Failed to update config file: %v.", err)
		}

		log.Println("✓ Export successful.")
		return
	}

	log.Printf("► Updating organization %s…", cfg.Organization)

	options := sync.Options{
		CreateMissingRepositories:  createRepositories,
		DeleteDanglingRepositories: deleteRepositories,
	}

	err = sync.Sync(ctx, cfg, client, options)
	if err != nil {
		log.Fatalf("⚠ Failed to sync state: %v.", err)
	}

	if confirm {
		log.Println("✓ Permissions successfully synchronized.")
	} else {
		log.Println("⚠ Run again with -confirm to apply the changes above.")
	}
}

func getToken() string {
	envName := "AQUAYMAN_TOKEN"
	token := os.Getenv(envName)
	if token == "" {
		log.Fatalf("⚠ No OAuth2 token specified in $%s.", envName)
	}

	return token
}
