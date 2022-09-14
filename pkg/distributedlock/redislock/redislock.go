package redislock

import (
	"context"
	"errors"
	"fmt"
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

	startAt  time.Time //start get lock time
	obtainAt time.Time //obtain lock and return success
	ttl      time.Duration

	mu     sync.Mutex
	closed bool
	stopCh chan struct{}
}

type LockOptions struct {
	token   string
	backoff backoffx.Backoffer
	//ttl   time.Duration
	minNetDelay time.Duration //the min network delay to redis server, default 2 time.milliseconde
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
	if lock.opt.minNetDelay == 0 {
		lock.opt.minNetDelay = time.Millisecond * 2
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

//ctx is for getting distributed lock
func (l *DisLock) Lock(ctx context.Context, ttl time.Duration) (ok bool, err error) {
	//避免当前锁崩溃后，锁永远不释放
	if ttl == 0 {
		return false, ErrNoExpiration
	}
	l.ttl = ttl

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(ttl))
		defer cancel()
	}

	defer l.opt.backoff.Reset()
	for {
		l.startAt = time.Now()
		ok, err = l.client.SetNX(ctx, l.key, l.opt.token, ttl).Result()
		if ok {
			l.obtainAt = time.Now()
			return
		}
		if isContextErr(err) {
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

//ctx is for task deadline, should renew lock ttl
func (l *DisLock) Run(ctx context.Context, task func(context.Context) error) error {
	ttl := l.lockTTL()
	if ttl < l.opt.minNetDelay {
		return fmt.Errorf("ttl(%d) < minNetDelay(%d)", ttl, l.opt.minNetDelay)
	}

	timer := &time.Timer{}
	var cancel context.CancelFunc
	expireAt := l.lockExpireAt()

	dl, ok := ctx.Deadline()
	if !ok {
		ctx, cancel = context.WithDeadline(ctx, expireAt)
		dl = expireAt
		defer cancel()
	} else if dl.After(expireAt) { //dl > expireAt, should renew at ttl / 2
		intvl := ttl / 2
		timer = time.NewTimer(intvl)
		defer timer.Stop()
	}
	//if dl < expireAt, don't need to renew lock key, no timer to refresh

	defer func() {
		nctx, _ := context.WithTimeout(context.Background(), time.Second)
		err := l.Release(nctx)
		if err != nil {
			fmt.Println(err)
		}
	}()
	defer l.opt.backoff.Reset()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.stopCh:
			return errors.New("closed")
		case <-timer.C:
			//come to here, means  expireAt < dl, so pttl to dl, because task maybe continue or block util ctx deadline
			pttl := time.Until(dl)
			if pttl < l.opt.minNetDelay {
				return fmt.Errorf("pttl(%d) < minNetDelay(%d)", ttl, l.opt.minNetDelay)
			}
			err := l.Refresh(ctx, pttl) //renew
			if err == nil {
				fmt.Println("refresh ok")
				continue
			}
			if err == ErrNotObtained {
				//key is expire ?
				return err
			}
			//err != nil
			ttl := time.Until(expireAt)
			if ttl < l.opt.minNetDelay {
				return fmt.Errorf("ttl(%d) < minNetDelay(%d)", ttl, l.opt.minNetDelay)
			}
			timer.Reset(ttl / 2) //next time to refresh(renew) lock

		default:
			err := task(ctx)
			if err == nil {
				return nil
			}
			time.Sleep(l.opt.backoff.Duration())
		}
	}
}

//因为网络有延迟的，所以在redis 的ttl 跟应用层计算出来时间是有差异的
func (l *DisLock) lockExpireAt() time.Time {
	return l.startAt.Add(l.obtainAt.Sub(l.startAt)/2 + l.ttl)
}

func (l *DisLock) lockTTL() time.Duration {
	return time.Until(l.lockExpireAt())
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

/*
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
			nctx, _ := context.WithTimeout(context.Background(), time.Second)
			l.Release(nctx)
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
*/
