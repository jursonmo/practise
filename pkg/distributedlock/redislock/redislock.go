package redislock

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/rfyiamcool/backoff"
)

var (
	luaRefresh = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`)
	luaRelease = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)
	luaPTTL    = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pttl", KEYS[1]) else return -3 end`)
)
var (
	ErrNoExpiration = errors.New("not allowed to lock key with no expiration")
	// ErrNotObtained is returned when a lock cannot be obtained.
	ErrNotObtained = errors.New("redislock: not obtained")

	// ErrLockNotHeld is returned when trying to release an inactive lock.
	ErrLockNotHeld = errors.New("redislock: lock not held")
)

var defaultBackoff = backoff.NewBackOff(backoff.WithMinDelay(time.Millisecond*5), backoff.WithMaxDelay(time.Millisecond*50),
	backoff.WithFactor(1.5), backoff.WithJitterFlag(true))

type DisLock struct {
	ctx    context.Context
	client *redis.Client
	key    string
	opt    LockOptions
	mu     sync.Mutex
	closed bool
	stopCh chan struct{}
}

type LockOptions struct {
	token   string
	backoff backoffx.Backoffer
	//ttl   time.Duration
}

type LockOption func(*LockOptions)

func NewDisLock(client *redis.Client, key string, opts ...LockOption) *DisLock {
	if client == nil || key == "" {
		return nil
	}
	lock := &DisLock{client: client, stopCh: make(chan struct{}, 1)}
	for _, opt := range opts {
		opt(&lock.opt)
	}
	if lock.opt.token == "" {
		lock.opt.token = uuid.New().String()
	}
	if lock.opt.backoff == nil {
		lock.opt.backoff = defaultBackoff
	}
	return lock
}

func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
func (l *DisLock) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed == true {
		return
	}
	l.closed = true
	close(l.stopCh)
}

//ctx deadline is task deadline, ttl is key time live, ttl 往往表示do 所花的时间，当然如果ttl 内，do 没有完成，会自动续约
//ctx task 的期限应该要比ttl 大，不然，task 期限到的时候要注意把锁释放掉
func (l *DisLock) Lock(ctx context.Context, ttl time.Duration, do func(context.Context) error, fail func(context.Context) error) (ok bool, err error) {
	//避免当前锁崩溃后，锁永远不释放
	if ttl == 0 {
		return false, ErrNoExpiration
	}

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(ttl))
		//defer cancel()
	}

	defer l.opt.backoff.Reset()
	defer func() {
		if err == nil {
			go l.execute(ctx, ttl, do)
			go l.autoRefresh(ctx)
		}
	}()

	for {
		ok, err = l.client.SetNX(ctx, l.key, l.opt.token, ttl).Result()
		if ok {
			return
		}
		if isContextErr(err) {
			cancel()
			return
		}
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		// backoff and retry to lock
		// err == nil && !ok
		time.Sleep(l.opt.backoff.Duration())
	}
}
func (l *DisLock) execute(ctx context.Context, ttl time.Duration, do func(context.Context) error) {
	defer l.Close()
	err := do(ctx)
	if err == nil || isContextErr(err) {
		return
	}

	timer := time.NewTimer(ttl / 3)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			//fail
			return
		case <-l.stopCh:
			return
		case <-timer.C:
			err := do(ctx)
			if err == nil || isContextErr(err) {
				return
			}
			timer.Reset(ttl / 3)
		}

	}
}
func (l *DisLock) autoRefresh(ctx context.Context) {
	dl, ok := ctx.Deadline()
	if !ok {
		panic("no deadline is set")
	}
	util := time.Until(dl)
	if util < time.Millisecond {
		return
	}
	pttl := util / 2

	intvl := pttl
	timer := time.NewTimer(intvl)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopCh:
			//do it once, if fail, key also expire in
			l.Release(ctx)
			return
		case <-timer.C:
			err := l.Refresh(ctx, pttl)
			if err == ErrNotObtained {
				//key is expire ?
				return
			}
			if err != nil {
				//try again
				intvl = intvl / 2
				if intvl < time.Millisecond {
					return
				}
				timer.Reset(intvl)
				continue
			}
			// err == nil ,refresh ok
			//如果续约成功，那么pttl 就会超过ctx 的deadline
		}
	}
}

func (l *DisLock) Refresh(ctx context.Context, ttl time.Duration) error {
	ttlVal := strconv.FormatInt(int64(ttl/time.Millisecond), 10)
	status, err := luaRefresh.Run(ctx, l.client, []string{l.key}, l.opt.token, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		return nil
	}
	// err == nil, result is 0, means key not exsit
	return ErrNotObtained
}

func (l *DisLock) Release(ctx context.Context) error {
	res, err := luaRelease.Run(ctx, l.client, []string{l.key}, l.opt.token).Result()
	if err == redis.Nil {
		return ErrLockNotHeld
	} else if err != nil {
		return err
	}

	if i, ok := res.(int64); !ok || i != 1 {
		return ErrLockNotHeld
	}
	return nil
}
