package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akeylesslabs/akeyless-go/v4"
	sdk "github.com/bitwarden/sdk-go"
	"github.com/spf13/viper"
)

func main() {
	ctx := context.Background()

	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Load configuration
	if *configPath != "" {
		viper.SetConfigFile(*configPath)
	} else if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		viper.SetConfigFile(envPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Initialize Akeyless client
	akClient := akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: []akeyless.ServerConfiguration{{
			URL: viper.GetString("akeyless.url"),
		}},
	}).V2Api

	// Authenticate with Akeyless
	authBody := akeyless.NewAuthWithDefaults()
	authBody.AccessId = akeyless.PtrString(viper.GetString("akeyless.access_id"))
	authBody.AccessKey = akeyless.PtrString(viper.GetString("akeyless.access_key"))

	var apiErr akeyless.GenericOpenAPIError
	authOut, _, err := akClient.Auth(ctx).Body(*authBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			log.Fatalf("Akeyless auth failed: %s", string(apiErr.Body()))
		}
		log.Fatalf("Akeyless auth failed: %v", err)
	}
	akToken := authOut.GetToken()

	// List all secrets from Akeyless
	listBody := akeyless.ListItems{
		Token: &akToken,
		Path:  akeyless.PtrString("/"),
		Type:  &[]string{"static-secret"},
	}
	listOut, _, err := akClient.ListItems(ctx).Body(listBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			log.Fatalf("Failed to list secrets: %s", string(apiErr.Body()))
		}
		log.Fatalf("Failed to list secrets: %v", err)
	}

	// Initialize Bitwarden client
	apiURL := viper.GetString("bitwarden.api_url")
	identityURL := viper.GetString("bitwarden.identity_url")
	bwClient, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		log.Fatalf("Failed to create Bitwarden client: %v", err)
	}

	// Authenticate with Bitwarden
	err = bwClient.AccessTokenLogin(viper.GetString("bitwarden.access_token"), nil)
	if err != nil {
		log.Fatalf("Bitwarden auth failed: %v", err)
	}

	orgID := viper.GetString("bitwarden.org_id")
	projectID := viper.GetString("bitwarden.project_id")

	fmt.Printf("Using Organization ID: %s\n", orgID)
	fmt.Printf("Using Project ID: %s\n", projectID)

	// Verify project exists
	project, err := bwClient.Projects().Get(projectID)
	if err != nil {
		log.Fatalf("Failed to access project %s: %v\nMake sure the project ID is correct and your access token has permission to access it.", projectID, err)
	}
	fmt.Printf("✓ Connected to project: %s\n\n", project.Name)

	// Get existing secrets from Bitwarden
	bwSecretsResp, err := bwClient.Secrets().List(orgID)
	if err != nil {
		log.Fatalf("Failed to list Bitwarden secrets: %v", err)
	}
	bwSecrets := make(map[string]string) // key -> secretID
	for _, s := range bwSecretsResp.Data {
		bwSecrets[s.Key] = s.ID
	}

	// Sync secrets
	secretGroups := make(map[string]map[string]string)
	syncNote := fmt.Sprintf("Synced from Akeyless @ %s", time.Now().Format(time.RFC3339))

	for _, item := range listOut.GetItems() {
		secretPath := item.GetItemName()
		fmt.Printf("Processing: %s\n", secretPath)

		// Get secret value from Akeyless
		gsvBody := akeyless.GetSecretValue{
			Names: []string{secretPath},
			Token: &akToken,
		}
		gsvOut, _, err := akClient.GetSecretValue(ctx).Body(gsvBody).Execute()
		if err != nil {
			log.Printf("Failed to get secret %s: %v", secretPath, err)
			continue
		}

		secretValue, ok := gsvOut[secretPath].(string)
		if !ok {
			log.Printf("Secret %s has non-string value, skipping", secretPath)
			continue
		}

		// Parse path and group nested secrets
		parts := strings.Split(strings.Trim(secretPath, "/"), "/")
		if len(parts) > 1 {
			parent := parts[0]
			child := strings.Join(parts[1:], "/")
			if secretGroups[parent] == nil {
				secretGroups[parent] = make(map[string]string)
			}
			secretGroups[parent][child] = secretValue
			fmt.Printf("✓ Grouped: %s -> %s.%s\n", secretPath, parent, child)
		} else {
			secretName := strings.Trim(secretPath, "/")
			if existingID, exists := bwSecrets[secretName]; exists {
				// Update existing secret
				for {
					_, err = bwClient.Secrets().Update(existingID, secretName, secretValue, syncNote, orgID, []string{projectID})
					if err != nil && strings.Contains(err.Error(), "429") {
						log.Printf("Rate limited, waiting 60 seconds...")
						time.Sleep(60 * time.Second)
						continue
					}
					break
				}
				if err != nil {
					log.Printf("Failed to update secret %s in Bitwarden: %v", secretName, err)
					continue
				}
				fmt.Printf("✓ Updated: %s\n", secretName)
			} else {
				// Create new secret
				for {
					_, err = bwClient.Secrets().Create(secretName, secretValue, syncNote, orgID, []string{projectID})
					if err != nil && strings.Contains(err.Error(), "429") {
						log.Printf("Rate limited, waiting 60 seconds...")
						time.Sleep(60 * time.Second)
						continue
					}
					break
				}
				if err != nil {
					log.Printf("Failed to create secret %s in Bitwarden: %v", secretName, err)
					continue
				}
				fmt.Printf("✓ Created: %s\n", secretName)
			}
		}
	}

	// Create or update grouped secrets as JSON
	for parent, children := range secretGroups {
		jsonValue, err := json.Marshal(children)
		if err != nil {
			log.Printf("Failed to marshal JSON for %s: %v", parent, err)
			continue
		}

		if existingID, exists := bwSecrets[parent]; exists {
			// Update existing grouped secret
			for {
				_, err = bwClient.Secrets().Update(existingID, parent, string(jsonValue), syncNote, orgID, []string{projectID})
				if err != nil && strings.Contains(err.Error(), "429") {
					log.Printf("Rate limited, waiting 60 seconds...")
					time.Sleep(60 * time.Second)
					continue
				}
				break
			}
			if err != nil {
				log.Printf("Failed to update grouped secret %s in Bitwarden: %v", parent, err)
				continue
			}
			fmt.Printf("✓ Updated grouped: %s (%d items)\n", parent, len(children))
		} else {
			// Create new grouped secret
			for {
				_, err = bwClient.Secrets().Create(parent, string(jsonValue), syncNote, orgID, []string{projectID})
				if err != nil && strings.Contains(err.Error(), "429") {
					log.Printf("Rate limited, waiting 60 seconds...")
					time.Sleep(60 * time.Second)
					continue
				}
				break
			}
			if err != nil {
				log.Printf("Failed to create grouped secret %s in Bitwarden: %v", parent, err)
				continue
			}
			fmt.Printf("✓ Created grouped: %s (%d items)\n", parent, len(children))
		}
	}

	fmt.Println("\nSync complete!")
}
