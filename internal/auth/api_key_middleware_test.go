package auth

import (
	"testing"
	"time"
)

func TestRouteAllowList(t *testing.T) {
	if !isRouteAllowed("/api/v1/articles/28", "/api/v1/articles/*") {
		t.Fatal("prefix should allow")
	}
	if isRouteAllowed("/api/v1/datasets", "/api/v1/articles/*") {
		t.Fatal("unrelated route should be denied")
	}
	if !isRouteAllowed("/api/v1/datasets", "*") {
		t.Fatal("wildcard should allow")
	}
}

func TestIPAllowList(t *testing.T) {
	if !isIPAllowed("192.168.0.150", "192.168.0.0/24") {
		t.Fatal("CIDR should allow")
	}
	if isIPAllowed("10.0.0.1", "192.168.0.0/24") {
		t.Fatal("CIDR should deny")
	}
}

func TestMemoryRateLimiter(t *testing.T) {
	l := NewMemoryRateLimiter(time.Minute)
	if ok, _ := l.Allow(1, 2); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := l.Allow(1, 2); !ok {
		t.Fatal("second request should pass")
	}
	if ok, retryAfter := l.Allow(1, 2); ok || retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("third request should be limited with retry-after, ok=%v retryAfter=%v", ok, retryAfter)
	}
	if ok, _ := l.Allow(2, 2); !ok {
		t.Fatal("different key should have separate bucket")
	}
	if ok, retryAfter := l.Allow(3, 0); !ok || retryAfter != 0 {
		t.Fatalf("zero limit should mean unlimited, ok=%v retryAfter=%v", ok, retryAfter)
	}
}
