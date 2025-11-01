package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	ctx := context.Background()

	cfg := LoadConfig()

	akeyless, err := NewAkeylessClient(ctx, cfg.Akeyless.URL, cfg.Akeyless.AccessID, cfg.Akeyless.AccessKey)
	if err != nil {
		log.Fatalf("Akeyless auth failed: %v", err)
	}

	fmt.Printf("Using Organization ID: %s\n", cfg.Bitwarden.OrgID)
	fmt.Printf("Using Project ID: %s\n", cfg.Bitwarden.ProjectID)

	bitwarden, err := NewBitwardenClient(
		cfg.Bitwarden.APIURL,
		cfg.Bitwarden.IdentityURL,
		cfg.Bitwarden.AccessToken,
		cfg.Bitwarden.OrgID,
		cfg.Bitwarden.ProjectID,
	)
	if err != nil {
		log.Fatalf("Bitwarden auth failed: %v", err)
	}

	syncer := NewSyncer(akeyless, bitwarden)
	if err := syncer.Sync(ctx); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	fmt.Println("\nSync complete!")
}
