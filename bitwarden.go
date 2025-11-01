package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	sdk "github.com/bitwarden/sdk-go"
)

type BitwardenClient struct {
	client    sdk.BitwardenClientInterface
	orgID     string
	projectID string
}

func NewBitwardenClient(apiURL, identityURL, accessToken, orgID, projectID string) (*BitwardenClient, error) {
	client, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		return nil, err
	}

	if err := client.AccessTokenLogin(accessToken, nil); err != nil {
		return nil, err
	}

	project, err := client.Projects().Get(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to access project %s: %w", projectID, err)
	}
	fmt.Printf("✓ Connected to project: %s\n\n", project.Name)

	return &BitwardenClient{
		client:    client,
		orgID:     orgID,
		projectID: projectID,
	}, nil
}

func (c *BitwardenClient) GetExistingSecrets() (map[string]*sdk.SecretIdentifierResponse, error) {
	resp, err := c.client.Secrets().List(c.orgID)
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]*sdk.SecretIdentifierResponse)
	for i := range resp.Data {
		secrets[resp.Data[i].Key] = &resp.Data[i]
	}
	return secrets, nil
}

func (c *BitwardenClient) GetSecret(id string) (*sdk.SecretResponse, error) {
	return c.client.Secrets().Get(id)
}

func (c *BitwardenClient) CreateSecret(key, value, note string) error {
	for {
		_, err := c.client.Secrets().Create(key, value, note, c.orgID, []string{c.projectID})
		if err != nil && strings.Contains(err.Error(), "429") {
			log.Printf("Rate limited, waiting 60 seconds...")
			time.Sleep(60 * time.Second)
			continue
		}
		return err
	}
}

func (c *BitwardenClient) UpdateSecret(id, key, value, note string) error {
	for {
		_, err := c.client.Secrets().Update(id, key, value, note, c.orgID, []string{c.projectID})
		if err != nil && strings.Contains(err.Error(), "429") {
			log.Printf("Rate limited, waiting 60 seconds...")
			time.Sleep(60 * time.Second)
			continue
		}
		return err
	}
}
