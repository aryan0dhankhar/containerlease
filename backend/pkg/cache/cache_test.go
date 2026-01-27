package cache

import (
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 1*time.Second)
	val, ok := c.Get("key1")
	if !ok || val != "value1" {
		t.Fatalf("expected value1, got %v, exists=%v", val, ok)
	}
}

func TestExpiration(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	_, ok := c.Get("key1")
	if ok {
		t.Fatalf("expected expired key to return false")
	}
}

func TestDelete(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 1*time.Second)
	c.Delete("key1")
	_, ok := c.Get("key1")
	if ok {
		t.Fatalf("expected deleted key to return false")
	}
}

func TestInvalidate(t *testing.T) {
	c := New()
	c.Set("container:1", "c1", 1*time.Second)
	c.Set("container:2", "c2", 1*time.Second)
	c.Set("snapshot:1", "s1", 1*time.Second)
	c.Invalidate("container:")
	_, ok1 := c.Get("container:1")
	_, ok2 := c.Get("container:2")
	_, ok3 := c.Get("snapshot:1")
	if ok1 || ok2 {
		t.Fatalf("expected container keys to be invalidated")
	}
	if !ok3 {
		t.Fatalf("expected snapshot:1 to still exist")
	}
}
