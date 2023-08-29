package client

import "github.com/jursonmo/practise/pkg/proto/session"

type Option func(*Client)

func WithOnDialFail(h func(error)) Option {
	return func(c *Client) {
		c.onDialFail = h
	}
}

func WithOnConnect(h func(session.Sessioner) error) Option {
	return func(c *Client) {
		c.onConnect = h
	}
}

func WithOnStop(h func(string)) Option {
	return func(c *Client) {
		c.onStop = h
	}
}
