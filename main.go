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
	"github.com/kubermatic-labs/aquayman/pkg/publisher"
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
		checkNames         = false
		exportMode         = false
		createRepositories = false
		deleteRepositories = false

		// Set this to enable vault integration; as the Vault API
		// client uses VAULT_ADDR and VAULT_TOKEN env vars already,
		// we simply do the same.
		enableVault = false
	)

	flag.StringVar(&configFile, "config", configFile, "path to the config.yaml")
	flag.BoolVar(&showVersion, "version", showVersion, "show the Aquayman version and exit")
	flag.BoolVar(&confirm, "confirm", confirm, "must be set to actually perform any changes on quay.io")
	flag.BoolVar(&validate, "validate", validate, "validate the given configuration syntax and then exit")
	flag.BoolVar(&checkNames, "check-names", checkNames, "(only with -validate) validate that users actually exist (requires valid quay.io credentials)")
	flag.BoolVar(&exportMode, "export", exportMode, "export quay.io state and update the config file (-config flag)")
	flag.BoolVar(&createRepositories, "create-repos", createRepositories, "create repositories listed in the config file but not existing on quay.io yet")
	flag.BoolVar(&deleteRepositories, "delete-repos", deleteRepositories, "delete repositories on quay.io that are not listed in the config file")
	flag.BoolVar(&enableVault, "enable-vault", enableVault, "enable Vault integration (VAULT_ADDR and VAULT_TOKEN env vars must be set also)")
	flag.Parse()

	if showVersion {
		fmt.Printf("Aquayman %s (built at %s)\n", version, date)
		return
	}

	if enableVault {
		if os.Getenv("VAULT_ADDR") == "" || os.Getenv("VAULT_TOKEN") == "" {
			log.Fatal("⚠ Both VAULT_ADDR and VAULT_TOKEN environment variables need to be set if -enable-vault is used.")
		}
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

	var (
		client *quay.Client
	)

	// validate config unless in export mode, where an incomplete
	// configuration is allowed and even expected
	if !exportMode {
		if checkNames {
			client, err = quay.NewClient(getToken(), 30*time.Second, true)
			if err != nil {
				log.Fatalf("⚠ Failed to create quay.io API client: %v.", err)
			}
		}

		if err := cfg.Validate(ctx, client); err != nil {
			log.Fatalf("Configuration is invalid: %v", err)
		}
	}

	if validate {
		log.Println("✓ Configuration is valid.")
		return
	}

	if client == nil {
		client, err = quay.NewClient(getToken(), 30*time.Second, !confirm)
		if err != nil {
			log.Fatalf("⚠ Failed to create quay.io API client: %v.", err)
		}
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

	var pub publisher.Publisher
	if enableVault {
		pub, err = publisher.NewVaultPublisher(cfg.Organization)
		if err != nil {
			log.Fatalf("⚠ Failed to create Vault client: %v.", err)
		}
	}

	log.Printf("► Updating organization %s…", cfg.Organization)

	options := sync.Options{
		CreateMissingRepositories:  createRepositories,
		DeleteDanglingRepositories: deleteRepositories,
		Publisher:                  pub,
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
