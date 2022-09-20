package session

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
)

type Sessioner interface {
	Add(context.Context, string, string, time.Time) error
	IsValid(context.Context, string, string) bool
	Close() error
}

type SessionManager struct {
	DB *redis.Client
}

func NewManager(DB *redis.Client) Sessioner {
	return SessionManager{DB: DB}
}

func (s SessionManager) Close() error {
	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("Could not close redisDB: %v", err)
	}

	return nil
}

func (s SessionManager) IsValid(ctx context.Context, userID string, jti string) bool {
	exists, err := s.DB.HExists(ctx, userID, jti).Result()
	if !exists || err != nil {
		return false
	}

	return true
}

func (s SessionManager) Add(ctx context.Context, userID string, jti string, expiresAt time.Time) error {
	hmap, err := s.DB.HGetAll(ctx, userID).Result()
	if err != nil {
		return err
	}

	for jti, exp := range hmap {
		if exp <= strconv.FormatInt(time.Now().Truncate(time.Second).Unix(), 10) {
			err := s.DB.HDel(ctx, userID, jti).Err()
			if err != nil {
				return err
			}
		}
	}

	err = s.DB.HSet(ctx, userID, jti, strconv.FormatInt(expiresAt.Unix(), 10)).Err()

	if err != nil {
		return err
	}

	return nil
}
