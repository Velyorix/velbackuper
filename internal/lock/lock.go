package lock

import "context"

type Locker interface {
	Acquire(ctx context.Context) error
	Release(ctx context.Context) error
}
