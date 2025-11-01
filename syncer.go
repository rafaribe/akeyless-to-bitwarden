package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	sdk "github.com/bitwarden/sdk-go"
)

type Syncer struct {
	akeyless  *AkeylessClient
	bitwarden *BitwardenClient
}

func NewSyncer(akeyless *AkeylessClient, bitwarden *BitwardenClient) *Syncer {
	return &Syncer{
		akeyless:  akeyless,
		bitwarden: bitwarden,
	}
}

func (s *Syncer) Sync(ctx context.Context) error {
	paths, err := s.akeyless.ListSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	fmt.Printf("Found %d secrets in Akeyless\n", len(paths))

	existing, err := s.bitwarden.GetExistingSecrets()
	if err != nil {
		return fmt.Errorf("failed to list Bitwarden secrets: %w", err)
	}

	timestamp := time.Now().Format(time.RFC3339)

	for _, path := range paths {
		if err := s.syncSecret(ctx, path, existing, timestamp); err != nil {
			log.Printf("Failed to sync %s: %v", path, err)
		}
	}

	return nil
}

func (s *Syncer) syncSecret(ctx context.Context, path string, existing map[string]*sdk.SecretIdentifierResponse, timestamp string) error {
	fmt.Printf("Processing: %s\n", path)

	value, err := s.akeyless.GetSecret(ctx, path)
	if err != nil {
		return err
	}

	key := strings.Trim(path, "/")

	if secret, exists := existing[key]; exists {
		fullSecret, err := s.bitwarden.GetSecret(secret.ID)
		if err != nil {
			return fmt.Errorf("failed to get secret details: %w", err)
		}
		note := fullSecret.Note
		if note == "" {
			note = fmt.Sprintf("Synced from Akeyless @ %s", timestamp)
		} else {
			note = fmt.Sprintf("%s | Modified @ %s", note, timestamp)
		}
		if err := s.bitwarden.UpdateSecret(secret.ID, key, value, note); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Printf("✓ Updated: %s\n", key)
	} else {
		note := fmt.Sprintf("Synced from Akeyless @ %s", timestamp)
		if err := s.bitwarden.CreateSecret(key, value, note); err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		fmt.Printf("✓ Created: %s\n", key)
	}

	return nil
}
