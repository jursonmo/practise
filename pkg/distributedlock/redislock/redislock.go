package redislock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/rfyiamcool/backoff"
)

type Log interface {
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	// Notice(format string, a ...interface{})
	// Warn(format string, a ...interface{})
	Errorf(format string, a ...interface{})
	Fatalf(format string, a ...interface{})
}

type mylog struct{}

func (l *mylog) Debugf(format string, a ...interface{}) {
	log.Printf(format, a...)
}
func (l *mylog) Infof(format string, a ...interface{}) {
	log.Printf(format, a...)
}
func (l *mylog) Errorf(format string, a ...interface{}) {
	log.Printf(format, a...)
}
func (l *mylog) Fatalf(format string, a ...interface{}) {
	log.Printf(format, a...)
}

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

func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

type DisLock struct {
	ctx    context.Context
	client *redis.Client
	key    string
	opt    LockOptions

	startAt  time.Time     //start get lock time
	obtainAt time.Time     //obtain lock and return success
	ttl      time.Duration //lock time expected

	mu     sync.Mutex
	closed bool
	stopCh chan struct{}
}

type LockOptions struct {
	log         Log
	token       string
	backoff     backoffx.Backoffer
	minNetDelay time.Duration //the min network delay to redis server, default 2 time.milliseconde
}

type LockOption func(*LockOptions)

func WithToken(token string) LockOption {
	return func(lo *LockOptions) {
		lo.token = token
	}
}
func WithBackoff(backoff backoffx.Backoffer) LockOption {
	return func(lo *LockOptions) {
		lo.backoff = backoff
	}
}
func WithMinNetDelay(networkDelay time.Duration) LockOption {
	return func(lo *LockOptions) {
		lo.minNetDelay = networkDelay
	}
}

func WithLog(log Log) LockOption {
	return func(lo *LockOptions) {
		lo.log = log
	}
}

func NewDisLock(client *redis.Client, key string, opts ...LockOption) *DisLock {
	if client == nil || key == "" {
		return nil
	}
	lock := &DisLock{client: client, key: key, stopCh: make(chan struct{}, 1)}

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
	if lock.opt.log == nil {
		lock.opt.log = (*mylog)(nil)
	}
	return lock
}

func (l *DisLock) String() string {
	return fmt.Sprintf("key:%s, token:%s, ttl:%v, startAt:%v, obtainAt:%v, net ttl:%v", l.key, l.opt.token, l.ttl, l.startAt, l.obtainAt, l.obtainAt.Sub(l.startAt))
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
	//ttl 不能为0, 避免当前锁崩溃后，锁永远不释放
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
		start := time.Now()
		ok, err = l.client.SetNX(ctx, l.key, l.opt.token, ttl).Result()
		if ok {
			l.startAt = start
			l.obtainAt = time.Now()
			return
		}
		if isContextErr(err) {
			return
		}
		if err != nil {
			l.opt.log.Errorf("setnx err:%v", err)
			time.Sleep(l.opt.backoff.Duration())
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
	l.opt.log.Infof("ttl:%v", ttl)

	refreshTimer := &time.Timer{}
	var cancel context.CancelFunc
	lockExpireAt := l.lockExpireAt()

	dl, ok := ctx.Deadline()
	if !ok {
		//如果没有设置过期时间，那么就用lock 的过期时间，task ctx 必须有超时机制，不能永久阻塞
		ctx, cancel = context.WithDeadline(ctx, lockExpireAt)
		dl = lockExpireAt
		defer cancel()
	} else if dl.After(lockExpireAt) { //dl > lockExpireAt, should renew at ttl / 2
		intvl := ttl / 2
		refreshTimer = time.NewTimer(intvl)
		defer refreshTimer.Stop()
	}
	//if dl <= lockExpireAt, don't need to renew lock key, no timer to refresh

	defer func() {
		err := l.release()
		if err != nil {
			l.opt.log.Errorf("release err:%v\n", err)
		}
	}()

	taskctx := ctx
	var ncancel context.CancelFunc
	if dl.After(lockExpireAt) {
		taskctx, ncancel = context.WithDeadline(ctx, lockExpireAt) //确保业务task 执行过程中, lock key 肯定没有超时的
		defer ncancel()
	}
	err := task(taskctx)
	if err == nil {
		return nil
	}

	d := l.opt.backoff.Duration()
	taskTimer := time.NewTimer(d)
	defer taskTimer.Stop()
	defer l.opt.backoff.Reset()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.stopCh:
			return errors.New("closed")
		case <-refreshTimer.C:
			//come to here, means lock key's lockExpireAt < dl, so pttl to dl, because task maybe continue or block util ctx deadline
			pttl := time.Until(dl)
			if pttl < l.opt.minNetDelay {
				return fmt.Errorf("pttl(%v) < minNetDelay(%v)", ttl, l.opt.minNetDelay)
			}
			//每次续约的最大值不能超过期望的ttl
			if pttl > l.ttl {
				pttl = l.ttl
			}
			err := l.Refresh(ctx, pttl) //renew
			if err == nil {
				l.opt.log.Debugf("refresh ok, pttl:%v, task deadline %v, %v", pttl, time.Until(dl), dl)
				refreshTimer.Reset(pttl / 2) //next time to refresh(renew) lock
				continue
			}
			if err == ErrNotObtained {
				//key is expire ?
				return err
			}

			//err != nil
			lockExpireAt = l.lockExpireAt()
			ttl := time.Until(lockExpireAt)
			if ttl < l.opt.minNetDelay {
				return fmt.Errorf("ttl(%d) < minNetDelay(%d)", ttl, l.opt.minNetDelay)
			}
			refreshTimer.Reset(ttl / 2) //next time to refresh(renew) lock
		case <-taskTimer.C:
			err := task(ctx)
			if err == nil {
				return nil
			}
			taskTimer.Reset(l.opt.backoff.Duration())
		}
	}
}

func (l *DisLock) release() (err error) {
	//分布式锁只有一个持有者，多试几次不会造成太大压力
	l.opt.backoff.Reset()
	defer l.opt.backoff.Reset()
	for i := 0; i < 3; i++ {
		ttl := l.lockTTL()
		if ttl < l.opt.minNetDelay {
			return fmt.Errorf("d(%d) < minNetDelay(%d)", ttl, l.opt.minNetDelay)
		}
		nctx, _ := context.WithTimeout(context.Background(), ttl)

		err = l.Release(nctx)
		if err == nil {
			return nil
		}
		if err == ErrLockNotHeld {
			return err
		}

		l.opt.log.Debugf("relase times:%d, ttl:%v, err:%v", i, ttl, err)
		time.Sleep(l.opt.backoff.Duration())
	}
	return err
}

//因为网络有延迟的，所以在redis 的ttl 跟应用层计算出来时间是有差异的
func (l *DisLock) lockExpireAt() time.Time {
	//return l.startAt.Add(l.obtainAt.Sub(l.startAt)/2 + l.ttl)
	//这样的话，由于有网络延迟, redis lock 超时的时间比计算的要长一点,这样可以确保在业务处理的过程中，锁是没有被释放的
	return l.startAt.Add(l.ttl)
}

func (l *DisLock) lockTTL() time.Duration {
	return time.Until(l.lockExpireAt())
}

func (l *DisLock) Refresh(ctx context.Context, ttl time.Duration) error {
	ttlVal := strconv.FormatInt(int64(ttl/time.Millisecond), 10)
	start := time.Now()
	status, err := luaRefresh.Run(ctx, l.client, []string{l.key}, l.opt.token, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		l.startAt = start
		l.obtainAt = time.Now()
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
