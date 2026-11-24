package cache

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	c := NewMemoryCache()
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

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
	c, err := New(ctx, "redis://admin:super-secret-password@127.0.0.1:1/0")
	if err == nil {
		if c != nil {
			_ = c.Close()
		}
		t.Fatalf("expected error connecting to unreachable redis, got nil")
	}
	if strings.Contains(err.Error(), "super-secret-password") {
		t.Fatalf("expected redis password to be redacted from error, got: %v", err)
	}
}

func TestNew_InvalidRedisURL_RedactsCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, "redis://user:topsecret@:badport")
	if err == nil {
		t.Fatalf("expected error for invalid redis URL")
	}
	if strings.Contains(err.Error(), "topsecret") {
		t.Fatalf("expected password to be redacted from parse error, got: %v", err)
	}
}

func TestMemoryCache_DefensiveCopyAndBoundedEviction(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCacheWithLimit(2)
	defer func() { _ = c.Close() }()

	orig := []byte("secret-value")
	if err := c.Set(ctx, "k1", orig, time.Minute); err != nil {
		t.Fatalf("Set k1 failed: %v", err)
	}
	// Mutating caller slice must not affect cached bytes.
	orig[0] = 'X'

	got, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get k1 failed: %v", err)
	}
	if string(got) != "secret-value" {
		t.Fatalf("expected defensive copy on Set, got %q", string(got))
	}

	// Mutating returned slice must not affect cached bytes.
	got[0] = 'Y'
	gotAgain, err := c.Get(ctx, "k1")
	if err != nil || string(gotAgain) != "secret-value" {
		t.Fatalf("expected defensive copy on Get, got %q (%v)", string(gotAgain), err)
	}

	// Fill capacity and verify oldest entry eviction.
	_ = c.Set(ctx, "k2", []byte("v2"), time.Minute)
	_ = c.Set(ctx, "k3", []byte("v3"), time.Minute)

	if _, err := c.Get(ctx, "k1"); err != ErrNotFound {
		t.Fatalf("expected oldest key k1 to be evicted, got err=%v", err)
	}
	if v, err := c.Get(ctx, "k2"); err != nil || string(v) != "v2" {
		t.Fatalf("expected k2 to remain cached, got %q (%v)", string(v), err)
	}
	if v, err := c.Get(ctx, "k3"); err != nil || string(v) != "v3" {
		t.Fatalf("expected k3 to be cached, got %q (%v)", string(v), err)
	}
}
