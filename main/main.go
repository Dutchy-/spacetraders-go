package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/time/rate"

	"github.com/Dutchy-/spacetrader-go/client"
)

//go:generate oapi-codegen --package=client -generate=types -o ./client/types.go https://stoplight.io/api/v1/projects/spacetraders/spacetraders/nodes/reference/SpaceTraders.json?fromExportButton=true&snapshotType=http_service&deref=optimizedBundle
//go:generate oapi-codegen --package=client -generate=client -o ./client/client.go https://stoplight.io/api/v1/projects/spacetraders/spacetraders/nodes/reference/SpaceTraders.json?fromExportButton=true&snapshotType=http_service&deref=optimizedBundle

const (
	API_URL    = "https://api.spacetraders.io/v2"
	TOKEN_FILE = "token"
)

var Token string

func AddBearer(ctx context.Context, req *http.Request) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", Token))
	return nil
}

type RLHTTPClient struct {
	client      *http.Client
	Ratelimiter *rate.Limiter
}

func (c *RLHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func NewClient(rl *rate.Limiter) *RLHTTPClient {
	c := &RLHTTPClient{
		client:      http.DefaultClient,
		Ratelimiter: rl,
	}
	return c
}

var Client client.ClientWithResponsesInterface

func main() {
	fmt.Println("starting client")
	b, err := os.ReadFile(TOKEN_FILE)
	if err != nil {
		panic(err)
	}
	Token = string(b)
	// Client, err = client.NewClientWithResponses(API_URL, client.WithHTTPClient(NewClient(rate.NewLimiter(2, 7))))
	Client, err = client.NewClientWithResponses(API_URL, client.WithRequestEditorFn(AddBearer), client.WithHTTPClient(NewClient(rate.NewLimiter(2, 7))))
	if err != nil {
		log.Fatalf("Failed to start client: %v\n", err)
	}

	game := NewGame()
	game.Run()

	// rresp, err := Client.RegisterWithResponse(context.TODO(), client.RegisterJSONRequestBody{
	// 	Faction: "QUANTUM",
	// 	Symbol:  "DVTCHY",
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// if rresp.StatusCode() == 201 {
	// 	regData := rresp.JSON201.Data
	// 	fmt.Printf("Register data: %+v\n", regData)
	// 	fmt.Printf("Token: %+v", regData.Token)
	// } else {
	// 	fmt.Println("Failed to register")
	// 	fmt.Printf("%s\n", rresp.Body)
	// }

}
