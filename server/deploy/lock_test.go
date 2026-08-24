package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireLockRefusesImmediatelyWhenHeld(t *testing.T) {
	root := t.TempDir()

	held, err := AcquireLock(root, 0)
	if err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	defer held.Release()

	if _, err := AcquireLock(root, 0); !errors.Is(err, ErrLocked) {
		t.Fatalf("second AcquireLock with wait=0: got %v, want ErrLocked", err)
	}
}

func TestAcquireLockGivesUpAfterTheWait(t *testing.T) {
	root := t.TempDir()

	held, err := AcquireLock(root, 0)
	if err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	defer held.Release()

	start := time.Now()
	_, err = AcquireLock(root, 60*time.Millisecond)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("waiting AcquireLock: got %v, want an error wrapping ErrLocked", err)
	}
	if waited := time.Since(start); waited < 60*time.Millisecond {
		t.Errorf("gave up after %v, before the requested wait", waited)
	}
}

func TestAcquireLockSucceedsOnceTheHolderReleases(t *testing.T) {
	root := t.TempDir()

	held, err := AcquireLock(root, 0)
	if err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	go func() {
		time.Sleep(80 * time.Millisecond)
		held.Release()
	}()

	lock, err := AcquireLock(root, 2*time.Second)
	if err != nil {
		t.Fatalf("waiting AcquireLock did not get the lock after release: %v", err)
	}
	lock.Release()
}

func TestReleaseThenReacquire(t *testing.T) {
	root := t.TempDir()

	lock, err := AcquireLock(root, 0)
	if err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	// Releasing twice must be harmless.
	if err := lock.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}

	again, err := AcquireLock(root, 0)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	again.Release()

	// The lock file stays behind by design; only the flock matters.
	if _, err := os.Stat(filepath.Join(root, LockFileName)); err != nil {
		t.Errorf("lock file missing after release: %v", err)
	}
}

func TestAcquireLockFailsWhenTheSiteRootIsMissing(t *testing.T) {
	_, err := AcquireLock(filepath.Join(t.TempDir(), "no-such-site"), 0)
	if err == nil {
		t.Fatal("expected an error for a missing site root")
	}
	if errors.Is(err, ErrLocked) {
		t.Fatalf("a missing site root is not a held lock: %v", err)
	}
}
