package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	envToken    = "SOURCEGRAPH_TOKEN"
	envEndpoint = "SOURCEGRAPH_ENDPOINT"
)

type client struct {
	token    string
	endpoint string
	client   *http.Client
}

func newClient(endpoint string, token string) (*client, error) {
	return &client{
		token:    token,
		endpoint: endpoint,
		client:   http.DefaultClient,
	}, nil
}

func (s *client) search(ctx context.Context, queryString string) (*apiResult, *metrics, error) {
	var body bytes.Buffer
	m := &metrics{}
	if err := json.NewEncoder(&body).Encode(map[string]interface{}{
		"query":     graphQLQuery,
		"variables": map[string]string{"query": queryString},
	}); err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.url(), ioutil.NopCloser(&body))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", "token "+s.token)
	req.Header.Set("X-Sourcegraph-Should-Trace", "true")

	start := time.Now()
	resp, err := s.client.Do(req)
	m.took = time.Since(start).Milliseconds()

	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		break
	default:
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	m.trace = resp.Header.Get("x-trace")

	// Decode the response.
	respDec := rawResult{Data: apiResult{}}
	if err := json.NewDecoder(resp.Body).Decode(&respDec); err != nil {
		return nil, nil, err
	}
	return &respDec.Data, m, nil
}

func (s *client) url() string {
	return s.endpoint + "/.api/graphql"
}
