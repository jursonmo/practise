package distributedlock

import "context"

type DistributedLocker interface {
	Lock(do func(context.Context) error) error
	Unlock(context.Context) error
}
