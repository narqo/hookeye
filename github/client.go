package github

import (
	"context"

	"github.com/machinebox/graphql"
)

type Client struct {
	c *graphql.Client

	ApiURL string
	Token  string
}

func NewClient(apiURL, token string, opts ...graphql.ClientOption) *Client {
	return &Client{
		c:      graphql.NewClient(apiURL, opts...),
		ApiURL: apiURL,
		Token:  token,
	}
}

func (c *Client) Run(ctx context.Context, req *graphql.Request, resp interface{}) error {
	if c.Token != "" {
		req.Header.Add("Authorization", "bearer "+c.Token)
	}
	return c.c.Run(ctx, req, resp)
}
