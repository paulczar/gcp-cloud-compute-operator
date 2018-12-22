package iam

import (
	"context"

	"golang.org/x/oauth2/google"
	iam "google.golang.org/api/iam/v1"
)

// Client is a placeholder for GCE stuff.
type Client struct {
	Service   *iam.Service
	ProjectID string
}

// New creates a new instance of GCECloud.
func New(project string) (*Client, error) {
	// Use oauth2.NoContext if there isn't a good context to pass in.
	ctx := context.TODO()
	client, err := google.DefaultClient(ctx, iam.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	c, err := iam.New(client)
	if err != nil {
		return nil, err
	}

	if project == "" {
		credentials, err := google.FindDefaultCredentials(ctx, iam.CloudPlatformScope)
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
