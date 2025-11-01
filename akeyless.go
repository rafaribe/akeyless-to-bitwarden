package main

import (
	"context"
	"errors"
	"log"

	"github.com/akeylesslabs/akeyless-go/v4"
)

type AkeylessClient struct {
	api   *akeyless.V2ApiService
	token string
}

func NewAkeylessClient(ctx context.Context, url, accessID, accessKey string) (*AkeylessClient, error) {
	api := akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: []akeyless.ServerConfiguration{{URL: url}},
	}).V2Api

	authBody := akeyless.NewAuthWithDefaults()
	authBody.AccessId = akeyless.PtrString(accessID)
	authBody.AccessKey = akeyless.PtrString(accessKey)

	var apiErr akeyless.GenericOpenAPIError
	authOut, _, err := api.Auth(ctx).Body(*authBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return nil, errors.New(string(apiErr.Body()))
		}
		return nil, err
	}

	return &AkeylessClient{
		api:   api,
		token: authOut.GetToken(),
	}, nil
}

func (c *AkeylessClient) ListSecrets(ctx context.Context) ([]string, error) {
	return c.listSecretsRecursive(ctx, "/")
}

func (c *AkeylessClient) listSecretsRecursive(ctx context.Context, path string) ([]string, error) {
	log.Printf("Listing path: %s", path)
	listBody := akeyless.ListItems{
		Token: &c.token,
		Path:  akeyless.PtrString(path),
	}

	var apiErr akeyless.GenericOpenAPIError
	listOut, _, err := c.api.ListItems(ctx).Body(listBody).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return nil, errors.New(string(apiErr.Body()))
		}
		return nil, err
	}

	var paths []string
	for _, item := range listOut.GetItems() {
		itemPath := item.GetItemName()
		itemType := item.GetItemType()
		log.Printf("Found: %s (type: %s)", itemPath, itemType)

		if itemType == "STATIC_SECRET" || itemType == "static-secret" {
			paths = append(paths, itemPath)
			// Also try to list inside this path in case it has nested secrets
			subPaths, err := c.listSecretsRecursive(ctx, itemPath)
			if err == nil && len(subPaths) > 0 {
				paths = append(paths, subPaths...)
			}
		} else if itemType == "FOLDER" || itemType == "folder" || itemType == "item" {
			subPaths, err := c.listSecretsRecursive(ctx, itemPath)
			if err != nil {
				log.Printf("Failed to list folder %s: %v", itemPath, err)
				continue
			}
			paths = append(paths, subPaths...)
		}
	}
	return paths, nil
}

func (c *AkeylessClient) GetSecret(ctx context.Context, path string) (string, error) {
	gsvBody := akeyless.GetSecretValue{
		Names: []string{path},
		Token: &c.token,
	}

	gsvOut, _, err := c.api.GetSecretValue(ctx).Body(gsvBody).Execute()
	if err != nil {
		return "", err
	}

	value, ok := gsvOut[path].(string)
	if !ok {
		log.Printf("Secret %s has non-string value, skipping", path)
		return "", errors.New("non-string value")
	}

	return value, nil
}
