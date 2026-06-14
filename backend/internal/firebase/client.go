package firebase

import (
	"context"

	firebasesdk "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

type Client struct {
	Auth      *auth.Client
	Firestore *firestore.Client
}

func New(ctx context.Context, credentialsFile, projectID string) (*Client, error) {
	var opts []option.ClientOption
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	app, err := firebasesdk.NewApp(ctx, &firebasesdk.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return nil, err
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{Auth: authClient, Firestore: fsClient}, nil
}
