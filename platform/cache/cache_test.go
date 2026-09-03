package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	c := NewMemoryCache()
	defer c.Close()

	ctx := context.Background()

	// 1. Not found
	_, err := c.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 2. Set and Get
	err = c.Set(ctx, "user:1", []byte("alice"), 1*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := c.Get(ctx, "user:1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(val) != "alice" {
		t.Fatalf("expected alice, got %s", string(val))
	}

	// 3. Delete
	err = c.Delete(ctx, "user:1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = c.Get(ctx, "user:1")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	c := NewMemoryCache()
	defer c.Close()

	ctx := context.Background()

	err := c.Set(ctx, "temp", []byte("data"), 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := c.Get(ctx, "temp")
	if err != nil || string(val) != "data" {
		t.Fatalf("expected data, got %s (%v)", string(val), err)
	}

	time.Sleep(30 * time.Millisecond)

	_, err = c.Get(ctx, "temp")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after expiry, got %v", err)
	}
}

func TestNew_EmptyURL_UsesMemory(t *testing.T) {
	ctx := context.Background()
	c, err := New(ctx, "")
	if err != nil {
		t.Fatalf("expected no error for empty URL, got %v", err)
	}
	defer c.Close()

	if err := c.Set(ctx, "foo", []byte("bar"), time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	val, err := c.Get(ctx, "foo")
	if err != nil || string(val) != "bar" {
		t.Fatalf("expected bar, got %s", string(val))
	}
}

func TestNew_UnreachableRedis_FailsClosed(t *testing.T) {
	ctx := context.Background()
	// Unreachable redis URL must return an error (fail closed) to prevent split brain
	c, err := New(ctx, "redis://127.0.0.1:1")
	if err == nil {
		if c != nil {
			c.Close()
		}
		t.Fatalf("expected error connecting to unreachable redis, got nil")
	}
}
