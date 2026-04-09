// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"strings"

	"github.com/deploymenttheory/go-api-sdk-jamfpro/sdk/jamfpro"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

type JamfConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	Client *jamfpro.Client
}

func NewJamfConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*JamfConnection, error) {
	conn := &JamfConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// Extract credentials and options from conf
	var clientID, clientSecret, instanceDomain string
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			clientID = cred.User
			clientSecret = string(cred.Secret)
		}
	}
	if domain, ok := conf.Options["instance_domain"]; ok {
		instanceDomain = domain
	}

	// Validate that all necessary credentials are provided
	if instanceDomain == "" || clientID == "" || clientSecret == "" {
		return nil, errors.New("missing required Jamf credentials: instance_domain, client_id, client_secret")
	}

	// Create the configuration container
	config := &jamfpro.ConfigContainer{
		LogLevel:       "warn",
		InstanceDomain: instanceDomain,
		AuthMethod:     "oauth2",
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}

	// Initialize the Jamf Pro client with the given configuration
	client, err := jamfpro.BuildClient(config)
	if err != nil {
		return nil, err
	}
	conn.Client = client
	log.Info().Msg("jamf> client initialized using BuildClient with ConfigContainer")

	return conn, nil
}

func (j *JamfConnection) Name() string {
	return "jamf"
}

func (j *JamfConnection) Asset() *inventory.Asset {
	return j.asset
}

func (j *JamfConnection) PlatformInfo() (*inventory.Platform, error) {
	return &inventory.Platform{
		Name:                  "jamf",
		Title:                 "Jamf Pro",
		Family:                []string{"jamf"},
		Kind:                  "api",
		Runtime:               "jamf",
		TechnologyUrlSegments: []string{"api", "jamf"},
	}, nil
}

func (j *JamfConnection) Identifier() string {
	domain := j.Conf.Options["instance_domain"]
	return "//platformid.api.mondoo.app/runtime/jamf/" + strings.ToLower(domain)
}
