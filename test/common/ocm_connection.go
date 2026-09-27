// developed with Cursor AI

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/openshift-online/ocm-sdk-go"
	configv1 "github.com/openshift/api/config/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultOCMURL = "https://api.stage.openshift.com"

type ocmCLIConfig struct {
	URL          string `json:"url"`
	TokenURL     string `json:"token_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// buildOCMConnection for A34 hive-managed quota patches.
func buildOCMConnection() (*sdk.Connection, error) {
	url := os.Getenv("OCM_URL")
	clientID := os.Getenv("OCM_CLIENT_ID")
	clientSecret := os.Getenv("OCM_CLIENT_SECRET")
	accessToken := os.Getenv("OCM_TOKEN")
	refreshToken := os.Getenv("OCM_REFRESH_TOKEN")
	tokenURL := ""

	if accessToken == "" && refreshToken == "" && clientID == "" {
		if cfg, err := loadOCMCLIConfig(); err == nil {
			if url == "" {
				url = cfg.URL
			}
			tokenURL = cfg.TokenURL
			clientID = cfg.ClientID
			clientSecret = cfg.ClientSecret
			accessToken = cfg.AccessToken
			refreshToken = cfg.RefreshToken
		}
	}

	if url == "" {
		url = defaultOCMURL
	}

	builder := sdk.NewConnectionBuilder().URL(url)
	if tokenURL != "" {
		builder = builder.TokenURL(tokenURL)
	}

	if clientID != "" {
		builder = builder.Client(clientID, clientSecret)
	}

	switch {
	case accessToken != "" && refreshToken != "":
		builder = builder.Tokens(accessToken, refreshToken)
	case accessToken != "":
		builder = builder.Tokens(accessToken)
	case refreshToken != "":
		builder = builder.Tokens(refreshToken)
	case clientID != "" && clientSecret != "":
		// Client credentials — SDK will request tokens on demand.
	default:
		return nil, fmt.Errorf(
			"OCM credentials required: log in with `ocm login` (local), or set OCM_TOKEN " +
				"(and OCM_REFRESH_TOKEN + OCM_CLIENT_ID), or OCM_CLIENT_ID + OCM_CLIENT_SECRET",
		)
	}

	connection, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("can't build OCM connection: %w", err)
	}

	return connection, nil
}

func loadOCMCLIConfig() (*ocmCLIConfig, error) {
	candidates := []string{}
	if p := os.Getenv("OCM_CONFIG"); p != "" {
		candidates = append(candidates, p)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".ocm.json"),
			filepath.Join(home, ".config", "ocm", "ocm.json"),
			filepath.Join(home, "Library", "Application Support", "ocm", "ocm.json"),
		)
	}

	var lastErr error
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		cfg := &ocmCLIConfig{}
		if err := json.Unmarshal(data, cfg); err != nil {
			lastErr = err
			continue
		}
		if cfg.AccessToken == "" && cfg.RefreshToken == "" && cfg.ClientSecret == "" {
			lastErr = fmt.Errorf("%s has no usable credentials", path)
			continue
		}
		return cfg, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ocm config not found")
	}
	return nil, lastErr
}

func resolveOCMClusterID(c k8sclient.Client, connection *sdk.Connection) (string, error) {
	if id := os.Getenv("CLUSTER_ID"); id != "" {
		return id, nil
	}

	cv := &configv1.ClusterVersion{}
	if err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: "version"}, cv); err != nil {
		return "", fmt.Errorf("CLUSTER_ID unset and failed to get ClusterVersion: %w", err)
	}
	externalID := string(cv.Spec.ClusterID)
	if externalID == "" {
		return "", fmt.Errorf("CLUSTER_ID unset and ClusterVersion.spec.clusterID is empty")
	}

	resp, err := connection.ClustersMgmt().V1().Clusters().List().
		Search(fmt.Sprintf("external_id is '%s'", externalID)).
		Size(1).
		Send()
	if err != nil {
		return "", fmt.Errorf("CLUSTER_ID unset and OCM lookup by external_id failed: %w", err)
	}
	if resp.Items() == nil || resp.Items().Len() == 0 {
		return "", fmt.Errorf("CLUSTER_ID unset and no OCM cluster found for external_id %s", externalID)
	}
	cluster := resp.Items().Get(0)
	if cluster == nil || cluster.ID() == "" {
		return "", fmt.Errorf("CLUSTER_ID unset and OCM cluster has empty id for external_id %s", externalID)
	}
	return cluster.ID(), nil
}
