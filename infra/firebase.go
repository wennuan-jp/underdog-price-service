package infra

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// FirebaseClient handles the connection to Firebase services
type FirebaseClient struct {
	App       *firebase.App
	Firestore *firestore.Client
}

// InitFirebase initializes the Firebase Admin SDK and returns a client
func InitFirebase(ctx context.Context, credentialsFile string) (*FirebaseClient, error) {
	// Use the service account credentials file
	opt := option.WithCredentialsFile(credentialsFile)
	
	// Initialize the Firebase App
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Initialize the Firestore client
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting firestore client: %v", err)
	}

	log.Println("Firebase Admin SDK initialized successfully with Cloud Firestore")
	
	return &FirebaseClient{
		App:       app,
		Firestore: client,
	}, nil
}

// Close closes the Firebase Firestore client
func (fc *FirebaseClient) Close() {
	if fc.Firestore != nil {
		if err := fc.Firestore.Close(); err != nil {
			log.Printf("Error closing Firestore client: %v", err)
		}
	}
}
