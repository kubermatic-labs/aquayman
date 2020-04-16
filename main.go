package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/kubermatic-labs/quayman/pkg/config"
	"github.com/kubermatic-labs/quayman/pkg/export"
	"github.com/kubermatic-labs/quayman/pkg/quay"
	"github.com/kubermatic-labs/quayman/pkg/sync"
)

func main() {
	ctx := context.Background()

	configFile := ""
	confirm := false
	validate := false
	exportMode := false

	flag.StringVar(&configFile, "config", configFile, "path to the config.yaml")
	flag.BoolVar(&confirm, "confirm", confirm, "must be set to actually perform any changes on quay.io")
	flag.BoolVar(&validate, "validate", validate, "validate the given configuration and then exit")
	flag.BoolVar(&exportMode, "export", exportMode, "export quay.io state and update the config file (-config flag)")
	flag.Parse()

	if configFile == "" {
		log.Fatal("⚠ No configuration (-config) specified.")
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

	if confirm && cfg.Organization == "kubermatic" {
		panic("nope")
	}

	log.Printf("► Updating organization %s…", cfg.Organization)

	err = sync.Sync(ctx, cfg, client)
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
	envName := "QUAYMAN_TOKEN"
	token := os.Getenv(envName)
	if token == "" {
		log.Fatalf("⚠ No OAuth2 token specified in $%s.", envName)
	}

	return token
}
