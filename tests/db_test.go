//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/JustUzair/irctc-microservice/utils"
	"github.com/JustUzair/irctc-microservice/utils/env"
)

type User struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name" gorm:"not null"`
	Email string `json:"email" gorm:"not null;uniqueIndex"`
}

func (User) TableName() string {
	return "temp_db_test"
}

func TestDatabaseAndRedisIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	t.Cleanup(cancel)

	config, err := env.Load()
	if err != nil {
		t.Fatalf("load test configuration: %v", err)
	}

	t.Log("connecting to PostgreSQL test database")
	db, err := utils.NewGormClient(
		ctx,
		config.TestDatabaseURL,
		&utils.PostgresGorm{
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
	)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Log("connected to PostgreSQL test database")

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get PostgreSQL connection pool: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	t.Log("connecting to Redis")
	redisClient, err := utils.NewRedisClient(ctx, config.RedisAddress, config.RedisPassword)
	if err != nil {
		t.Fatalf("connect to Redis: %v", err)
	}
	t.Log("connected to Redis")
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	user := User{
		Name:  "Integration Test User",
		Email: "integration-test@example.com",
	}
	redisKey := fmt.Sprintf("integration:user:%d", time.Now().UnixNano())

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		t.Log("cleanup: deleting cached test user from Redis")
		if err := redisClient.Del(cleanupCtx, redisKey).Err(); err != nil {
			t.Errorf("delete test user from Redis: %v", err)
		}
		if user.ID != 0 {
			t.Logf("cleanup: deleting test user %d from PostgreSQL", user.ID)
			if err := db.WithContext(cleanupCtx).Delete(&user).Error; err != nil {
				t.Errorf("delete test user from PostgreSQL: %v", err)
			}
		}
		t.Log("cleanup: dropping temp_db_test")
		if err := db.WithContext(cleanupCtx).Migrator().DropTable(&User{}); err != nil {
			t.Errorf("drop temporary test table: %v", err)
		}
		t.Log("cleanup complete")
	})

	t.Log("removing any stale temp_db_test table")
	if err := db.WithContext(ctx).Migrator().DropTable(&User{}); err != nil {
		t.Fatalf("remove stale temporary test table: %v", err)
	}
	t.Log("creating temp_db_test table")
	if err := db.WithContext(ctx).AutoMigrate(&User{}); err != nil {
		t.Fatalf("create temporary test table: %v", err)
	}
	t.Log("inserting test user")
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Logf("inserted test user with ID %d", user.ID)

	var persistedUser User
	t.Log("reading inserted user from PostgreSQL")
	if err := db.WithContext(ctx).First(&persistedUser, user.ID).Error; err != nil {
		t.Fatalf("read inserted test user: %v", err)
	}

	encodedUser, err := json.Marshal(persistedUser)
	if err != nil {
		t.Fatalf("encode test user for Redis: %v", err)
	}
	t.Log("storing test user in Redis")
	if err := redisClient.Set(ctx, redisKey, encodedUser, time.Minute).Err(); err != nil {
		t.Fatalf("store test user in Redis: %v", err)
	}

	t.Log("reading test user from Redis")
	cachedUserJSON, err := redisClient.Get(ctx, redisKey).Bytes()
	if err != nil {
		t.Fatalf("read test user from Redis: %v", err)
	}

	var cachedUser User
	if err := json.Unmarshal(cachedUserJSON, &cachedUser); err != nil {
		t.Fatalf("decode cached test user: %v", err)
	}
	if !reflect.DeepEqual(cachedUser, persistedUser) {
		t.Fatalf("cached user does not match database user: got %+v, want %+v", cachedUser, persistedUser)
	}
	t.Log("verified Redis user matches PostgreSQL user")
}
