package gcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/databricks/sdk-go/core/auth/internal"
	"github.com/databricks/sdk-go/core/client"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

type GoogleCredentials struct {
	GoogleCredentials string `name:"google_credentials" env:"GOOGLE_CREDENTIALS" auth:"sensitive"`
}

func (c GoogleCredentials) Name() string {
	return "google-accounts"
}

func (c GoogleCredentials) Configure(ctx context.Context, conf *client.ApiClient) (func(*http.Request) error, error) {
	host := internal.Host(conf.Config.Host)
	if c.GoogleCredentials == "" || !host.IsGcp() {
		return nil, nil
	}
	json, err := readCredentials(c.GoogleCredentials)
	if err != nil {
		err = fmt.Errorf("could not read GoogleCredentials. "+
			"Make sure the file exists, or the JSON content is valid: %w", err)
		return nil, err
	}
	inner, err := idtoken.NewTokenSource(ctx, conf.Config.Host, option.WithCredentialsJSON(json))
	if err != nil {
		return nil, fmt.Errorf("could not obtain OIDC token from JSON: %w", err)
	}
	// Obtain token source for creating Google Cloud Platform token.
	creds, err := google.CredentialsFromJSON(ctx, json,
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/compute")
	if err != nil {
		return nil, fmt.Errorf("could not obtain OAuth2 token from JSON: %w", err)
	}
	log.Printf("[INFO] Using Google Credentials")
	return internal.ServiceToServiceVisitor(inner, creds.TokenSource, "X-Databricks-GCP-SA-Access-Token"), nil
}

// Reads credentials as JSON. Credentials can be either a path to JSON file,
// or actual JSON string.
func readCredentials(credentials string) ([]byte, error) {
	// Try to read credentials as file path.
	if _, err := os.Stat(credentials); err == nil {
		jsonContents, err := ioutil.ReadFile(credentials)
		if err != nil {
			return jsonContents, err
		}
		return jsonContents, nil
	}
	// Assume that credential is actually JSON string.
	return []byte(credentials), nil
}
