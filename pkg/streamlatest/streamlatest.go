package streamlatest

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
)

type StreamLatest struct {
	sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	streamName  string
	addr        string
	redisClient *redis.Client
}

func NewStreamLatest(ctx context.Context, stream, addr string) *StreamLatest {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx, cancel := context.WithCancel(ctx)
	return &StreamLatest{ctx: ctx, cancel: cancel, streamName: stream, addr: addr, redisClient: client}
}

func (s *StreamLatest) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *StreamLatest) GetLatestMsg() (*redis.XMessage, error) {
	msgs, err := s.redisClient.XRevRangeN(s.ctx, s.streamName, "+", "-", 1).Result()
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return &msgs[0], nil
}

func (s *StreamLatest) ListenFromMsg(m *redis.XMessage) (*redis.XMessage, error) {
	fromId := "0"
	if m != nil {
		fromId = m.ID
	}
	for {
		streamsMsg, err := s.redisClient.XRead(s.ctx, &redis.XReadArgs{
			Streams: []string{s.streamName, fromId},
			Block:   0,
			Count:   4,
		}).Result()
		if err != nil {
			return nil, err
		}
		if n := len(streamsMsg); n > 0 {
			if nn := len(streamsMsg[n-1].Messages); nn > 0 {
				latest := streamsMsg[n-1].Messages[nn-1]
				return &latest, nil
			}
		}
	}
}
