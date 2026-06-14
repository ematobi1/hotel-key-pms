package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	defaultTTL     = 30 * time.Second
	lockKeyPrefix  = "lock:reservation:"
)

// ErrLockHeld is returned when SETNX fails because another process holds the lock.
var ErrLockHeld = fmt.Errorf("resource locked by another process")

type Locker interface {
	// Acquire attempts to acquire the lock for roomID over the date range.
	// Returns ErrLockHeld immediately if the lock is taken (no retry — the caller
	// must surface a 409 Conflict to the client).
	Acquire(ctx context.Context, roomID string, arrival, departure time.Time) (release func(context.Context), err error)
}

type redisLocker struct {
	rdb *redis.Client
	log *zap.Logger
}

func NewRedisLocker(rdb *redis.Client, log *zap.Logger) Locker {
	return &redisLocker{rdb: rdb, log: log}
}

func (l *redisLocker) Acquire(ctx context.Context, roomID string, arrival, departure time.Time) (func(context.Context), error) {
	key := fmt.Sprintf("%s%s:%s:%s",
		lockKeyPrefix,
		roomID,
		arrival.Format("20060102"),
		departure.Format("20060102"),
	)

	// SETNX — set if not exists; provides atomic mutex semantics for double-booking prevention
	ok, err := l.rdb.SetNX(ctx, key, "1", defaultTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("redis setnx: %w", err)
	}
	if !ok {
		return nil, ErrLockHeld
	}

	l.log.Debug("lock acquired", zap.String("key", key))

	release := func(releaseCtx context.Context) {
		if err := l.rdb.Del(releaseCtx, key).Err(); err != nil {
			l.log.Warn("failed to release lock", zap.String("key", key), zap.Error(err))
		}
	}
	return release, nil
}
