package prometheus

import (
	"PrometheusF6005/ont"
	"errors"
	"testing"
	"time"
)

func TestLoadWithReloginWaitsBeforeRefreshingSession(t *testing.T) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	collector := NewONTCollector("http://ont", "user", "password", time.Minute)
	collector.now = func() time.Time { return now }

	loginCount := 0
	collector.loginFn = func(endpoint, username, password string) (*ont.Session, error) {
		loginCount++
		return &ont.Session{Endpoint: endpoint}, nil
	}

	loadCount := 0
	load := func(*ont.Session) (*int, error) {
		loadCount++
		if loadCount == 1 {
			return nil, errors.New("session timeout")
		}
		result := 42
		return &result, nil
	}

	if _, err := loadWithRelogin(collector, load); err == nil {
		t.Fatal("expected the first load to report the expired session")
	}
	if _, err := loadWithRelogin(collector, load); err == nil {
		t.Fatal("expected login cooldown to reject an early retry")
	}
	if loginCount != 1 {
		t.Fatalf("expected one login during cooldown, got %d", loginCount)
	}

	now = now.Add(time.Minute)
	value, err := loadWithRelogin(collector, load)
	if err != nil {
		t.Fatalf("load after cooldown returned an error: %v", err)
	}
	if *value != 42 {
		t.Fatalf("unexpected value: %d", *value)
	}
	if loginCount != 2 {
		t.Fatalf("expected two logins, got %d", loginCount)
	}
	if loadCount != 2 {
		t.Fatalf("expected two load attempts, got %d", loadCount)
	}
}

func TestLoadWithReloginReturnsLoginError(t *testing.T) {
	collector := NewONTCollector("http://ont", "user", "password", time.Minute)
	expected := errors.New("ONT unavailable")
	collector.loginFn = func(endpoint, username, password string) (*ont.Session, error) {
		return nil, expected
	}

	_, err := loadWithRelogin(collector, func(*ont.Session) (*int, error) {
		t.Fatal("load must not run when login fails")
		return nil, nil
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
