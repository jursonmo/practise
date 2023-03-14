package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ConfigError struct {
	Code    int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Reason  string `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
	Message string `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

func (e *ConfigError) Error() string {
	return ""
}

type Client struct {
	rawUrl string
	url    url.URL
	ClientOpt
	client http.Client
}

type ClientOpts func(*ClientOpt)

type ClientOpt struct {
	timeout time.Duration
}

var DefaultClient *Client
var DefaultConfApi = "http://127.0.0.1:8080"

func init() {
	DefaultClient, _ = NewClient(DefaultConfApi)
}

func NewClient(u string, opts ...ClientOpts) (*Client, error) {
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	c := &Client{rawUrl: u, url: *url}
	c.client.Timeout = 2 * time.Second
	//c.client.Transport = DefaultTransport()
	for _, opt := range opts {
		opt(&c.ClientOpt)
	}
	return c, nil
}

func (c *Client) Request(ctx context.Context, reqObject interface{}, respObject interface{}, api string, method string) error {
	var reader io.Reader
	if reqObject != nil {
		data, err := json.Marshal(reqObject)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, api, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		if err == nil {
			e := new(ConfigError)
			if err = json.Unmarshal(data, e); err == nil {
				e.Code = int32(resp.StatusCode)
				e.Message = http.StatusText(resp.StatusCode)
				return e
			}
		}
		return err
	}

	if method == http.MethodGet && respObject != nil {
		err = json.Unmarshal(data, respObject)
		if err != nil {
			return err
		}
	}

	return nil
}
