package client

import (
	"net/http"
	"time"

	"github.com/alexruf/tankerkoenig-go"
)

// Client is a client for the Tankerkoenig API.
type Client = tankerkoenig.Client

// New returns a new Tankerkoenig API client that uses the given API key for
// authentication.
func New(apiKey string) *Client {
	return tankerkoenig.NewClient(apiKey, &http.Client{
		Timeout: time.Second * 15,
	})
}
