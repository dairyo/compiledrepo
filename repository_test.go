package compiledrepo_test

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dairyo/compiledrepo"
)

type mockLoader struct {
	loadCount int32
	data      map[string][]byte
}

func (m *mockLoader) Load(ctx context.Context, id string) ([]byte, error) {
	atomic.AddInt32(&m.loadCount, 1)
	data, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return data, nil
}

func (m *mockLoader) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for id := range m.data {
			if !yield(id, nil) {
				return
			}
		}
	}
}

type preloaderWrapper struct {
	loader *mockLoader
}

func (pw preloaderWrapper) All(ctx context.Context) iter.Seq2[string, error] {
	return pw.loader.All(ctx)
}

func TestRepository_Get(t *testing.T) {
	loader := &mockLoader{
		data: map[string][]byte{
			"res1": []byte("value1"),
		},
	}
	compiler := func(b []byte) (string, error) {
		return string(b), nil
	}

	repo := compiledrepo.NewRepository(loader, compiler)
	ctx := context.Background()

	// 1. Cache Miss
	val, err := repo.Get(ctx, "res1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}
	if atomic.LoadInt32(&loader.loadCount) != 1 {
		t.Errorf("expected load count 1, got %d", loader.loadCount)
	}

	// 2. Cache Hit
	val, err = repo.Get(ctx, "res1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}
	if atomic.LoadInt32(&loader.loadCount) != 1 {
		t.Errorf("expected load count 1, got %d", loader.loadCount)
	}
}

func TestRepository_ConcurrentGet(t *testing.T) {
	loader := &mockLoader{
		data: map[string][]byte{
			"res1": []byte("value1"),
		},
	}
	compiler := func(b []byte) (string, error) {
		return string(b), nil
	}

	repo := compiledrepo.NewRepository(loader, compiler)
	ctx := context.Background()

	var wg sync.WaitGroup
	const concurrency = 10
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.Get(ctx, "res1")
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&loader.loadCount) != 1 {
		t.Errorf("expected load count 1 due to singleflight, got %d", loader.loadCount)
	}
}

func TestRepository_Preload(t *testing.T) {
	loader := &mockLoader{
		data: map[string][]byte{
			"res1": []byte("value1"),
			"res2": []byte("value2"),
		},
	}
	compiler := func(b []byte) (string, error) {
		return string(b), nil
	}

	repo := compiledrepo.NewRepository(loader, compiler)
	ctx := context.Background()

	pw := preloaderWrapper{loader: loader}
	err := repo.Preload(ctx, pw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all are cached
	if _, err := repo.Get(ctx, "res1"); err != nil {
		t.Errorf("res1 should be cached: %v", err)
	}
	if _, err := repo.Get(ctx, "res2"); err != nil {
		t.Errorf("res2 should be cached: %v", err)
	}
}

func TestRepository_Snapshot(t *testing.T) {
	loader := &mockLoader{
		data: map[string][]byte{
			"res1": []byte("value1"),
		},
	}
	compiler := func(b []byte) (string, error) {
		return string(b), nil
	}

	repo := compiledrepo.NewRepository(loader, compiler)
	ctx := context.Background()

	_, _ = repo.Get(ctx, "res1")
	registry := repo.Snapshot()

	val, err := registry.Get("res1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	// Test mutation of repository does not affect registry
	loader.data["res2"] = []byte("value2")
	_, _ = repo.Get(ctx, "res2")
	
	val, err = registry.Get("res2")
	if err != nil {
		if err != compiledrepo.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	} else {
		t.Errorf("unexpectedly found res2 in registry")
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	// Use a simple string type to avoid nil issues
	repo := compiledrepo.NewRepository[string](nil, nil)
	registry := repo.Snapshot()
	
	val, err := registry.Get("none")
	if err != nil {
		if err != compiledrepo.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	} else {
		t.Errorf("unexpectedly found value for 'none'")
	}
	_ = val
}
