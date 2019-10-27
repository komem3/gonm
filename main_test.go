package gonm

import (
	"context"
	"testing"

	"cloud.google.com/go/datastore"
)

var testDsClient *datastore.Client

func TestMain(m *testing.M) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, "")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	testDsClient = client

	m.Run()
}
