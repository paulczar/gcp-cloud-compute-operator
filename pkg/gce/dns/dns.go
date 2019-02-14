package dns

import (
	"context"

	"golang.org/x/oauth2/google"
	dns "google.golang.org/api/dns/v1"
)

// Client is a placeholder for GCE stuff.
type Client struct {
	Service   *dns.Service
	ProjectID string
}

// CreateGCECloud creates a new instance of GCECloud.
func New(project string) (*Client, error) {
	// Use oauth2.NoContext if there isn't a good context to pass in.
	ctx := context.TODO()
	client, err := google.DefaultClient(ctx, dns.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	c, err := dns.New(client)
	if err != nil {
		return nil, err
	}

	if project == "" {
		credentials, err := google.FindDefaultCredentials(ctx, dns.CloudPlatformScope)
		if err != nil {
			return nil, err
		}
		project = credentials.ProjectID
	}
	// TODO validate project and network exist
	return &Client{
		Service:   c,
		ProjectID: project,
	}, nil
}
