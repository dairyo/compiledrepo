package compiledrepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// MockReader is a simple ReadCloser for testing.
type MockReader struct {
	io.Reader
	closed bool
}

func (m *MockReader) Close() error {
	m.closed = true
	return nil
}

func (m *MockReader) IsClosed() bool {
	return m.closed
}

// MockKeyIterator implements KeyIterator for testing.
type MockKeyIterator[K comparable] struct {
	keys []K
	err  error
}

func (m *MockKeyIterator[K]) All(ctx context.Context) iter.Seq2[K, error] {
	return func(yield func(K, error) bool) {
		for _, k := range m.keys {
			if !yield(k, m.err) {
				return
			}
		}
	}
}

func TestRepository_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessCases", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)

		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		key := "test-key"
		want := "compiled-value"
		reader := &MockReader{}

		// First access: should trigger Open and Compile
		mockOpener.EXPECT().Open(ctx, key).Return(reader, nil).Times(1)
		mockCompiler.EXPECT().Compile(ctx, reader).Return(want, nil).Times(1)

		got, err := repo.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.True(t, reader.IsClosed())

		// Second access: should return from cache, NO Open or Compile calls
		got2, err2 := repo.Get(ctx, key)
		require.NoError(t, err2)
		assert.Equal(t, want, got2)
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)

		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		key := "concurrent-key"
		want := "concurrent-value"
		reader := &MockReader{}

		// Even with concurrent requests, Open and Compile should only be called once
		mockOpener.EXPECT().Open(ctx, key).Return(reader, nil).Times(1)
		mockCompiler.EXPECT().Compile(ctx, reader).Return(want, nil).Times(1)

		var wg sync.WaitGroup
		numRequests := 10
		results := make(chan string, numRequests)
		errs := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				val, err := repo.Get(ctx, key)
				if err != nil {
					errs <- err
				} else {
					results <- val
				}
			}()
		}

		wg.Wait()
		close(results)
		close(errs)

		assert.Equal(t, 0, len(errs), "should have no errors")
		assert.Equal(t, numRequests, len(results))
		for res := range results {
			assert.Equal(t, want, res)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name       string
			key        string
			setupMocks func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string])
			wantErr    error
			wantErrMsg string
		}{
			{
				name: "Open failure returns context error",
				key:  "err-key-open",
				setupMocks: func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string]) {
					op.EXPECT().Open(ctx, "err-key-open").Return((*MockReader)(nil), fmt.Errorf("disk error")).Times(1)
				},
				wantErrMsg: "failed to open resource: disk error",
			},
			{
				name: "Compile failure returns context error",
				key:  "err-key-compile",
				setupMocks: func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string]) {
					reader := &MockReader{}
					op.EXPECT().Open(ctx, "err-key-compile").Return(reader, nil).Times(1)
					cp.EXPECT().Compile(ctx, reader).Return("", fmt.Errorf("syntax error")).Times(1)
				},
				wantErrMsg: "failed to compile resource: syntax error",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockOpener := NewMockOpener[string, *MockReader](ctrl)
				mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
				repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

				tt.setupMocks(mockOpener, mockCompiler)

				_, err := repo.Get(ctx, tt.key)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			})
		}
	})

	t.Run("PanicRecovery", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)

		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		key := "panic-key"

		mockOpener.EXPECT().Open(ctx, key).DoAndReturn(func(ctx context.Context, key string) (*MockReader, error) {
			panic("something went wrong")
		})

		_, err := repo.Get(ctx, key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compilation panicked: something went wrong")
	})
}

func TestRepository_Preload(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessCases", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		keys := []string{"k1", "k2", "k3"}
		it := &MockKeyIterator[string]{keys: keys}

		for _, k := range keys {
			reader := &MockReader{}
			mockOpener.EXPECT().Open(ctx, k).Return(reader, nil).Times(1)
			mockCompiler.EXPECT().Compile(ctx, reader).Return("val-"+k, nil).Times(1)
		}

		err := repo.Preload(ctx, it)
		require.NoError(t, err)

		// Verify all are cached
		for _, k := range keys {
			val, err := repo.Get(ctx, k)
			require.NoError(t, err)
			assert.Equal(t, "val-"+k, val)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name       string
			keys       []string
			iterErr    error
			setupMocks func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string], keys []string)
			wantErrMsg string
			useNilIter bool
		}{
			{
				name:    "Iterator returns error",
				keys:    []string{"k1", "k2"},
				iterErr: errors.New("iterator failure"),
				setupMocks: func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string], keys []string) {
					// Should not be called or only called until error
				},
				wantErrMsg: "preload iteration failed: iterator failure",
			},
			{
				name: "Get returns error for a key",
				keys: []string{"k1", "k2"},
				setupMocks: func(op *MockOpener[string, *MockReader], cp *MockCompiler[*MockReader, string], keys []string) {
					reader := &MockReader{}
					op.EXPECT().Open(ctx, "k1").Return(reader, nil).Times(1)
					cp.EXPECT().Compile(ctx, reader).Return("", fmt.Errorf("compile error")).Times(1)
				},
				wantErrMsg: "preload failed for key k1: failed to compile resource: compile error",
			},
			{
				name:       "Nil iterator returns error",
				useNilIter: true,
				wantErrMsg: "iterator error: iterator is nil",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockOpener := NewMockOpener[string, *MockReader](ctrl)
				mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
				repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

				var it KeyIterator[string]
				if !tt.useNilIter {
					it = &MockKeyIterator[string]{keys: tt.keys, err: tt.iterErr}
				}
				if tt.setupMocks != nil {
					tt.setupMocks(mockOpener, mockCompiler, tt.keys)
				}

				err := repo.Preload(ctx, it)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			})
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		cancelCtx, cancel := context.WithCancel(ctx)

		keys := []string{"k1", "k2", "k3"}

		// Setup: k1 succeeds, then cancel context
		reader1 := &MockReader{}
		mockOpener.EXPECT().Open(cancelCtx, "k1").Return(reader1, nil).Times(1)
		mockCompiler.EXPECT().Compile(cancelCtx, reader1).Return("val1", nil).Times(1)

		// We can't easily cancel exactly between iterations in this simple MockKeyIterator
		// unless we add a hook. Let's use a custom iterator.
		customIt := &customIterator{
			keys: keys,
			onNext: func(k string) {
				if k == "k1" {
					cancel()
				}
			},
		}

		err := repo.Preload(cancelCtx, customIt)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
	})
}

type customIterator struct {
	keys   []string
	onNext func(string)
}

func (c *customIterator) All(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, k := range c.keys {
			if c.onNext != nil {
				c.onNext(k)
			}
			if !yield(k, nil) {
				return
			}
		}
	}
}

func TestRepository_Snapshot(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessCases", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		items := map[string]string{
			"k1": "v1",
			"k2": "v2",
		}

		for k, v := range items {
			reader := &MockReader{}
			mockOpener.EXPECT().Open(ctx, k).Return(reader, nil).Times(1)
			mockCompiler.EXPECT().Compile(ctx, reader).Return(v, nil).Times(1)
			_, err := repo.Get(ctx, k)
			require.NoError(t, err)
		}

		registry := repo.Snapshot()

		for k, v := range items {
			got, ok := registry.Get(k)
			assert.True(t, ok, "key %s should be present", k)
			assert.Equal(t, v, got)
		}
	})

	t.Run("ImmutabilityCases", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOpener := NewMockOpener[string, *MockReader](ctrl)
		mockCompiler := NewMockCompiler[*MockReader, string](ctrl)
		repo := NewRepository[string, *MockReader, string](mockOpener, mockCompiler)

		// Initial state
		k1, v1 := "k1", "v1"
		reader1 := &MockReader{}
		mockOpener.EXPECT().Open(ctx, k1).Return(reader1, nil).Times(1)
		mockCompiler.EXPECT().Compile(ctx, reader1).Return(v1, nil).Times(1)
		_, err := repo.Get(ctx, k1)
		require.NoError(t, err)

		// Take snapshot
		registry := repo.Snapshot()

		// Add new item after snapshot
		k2, v2 := "k2", "v2"
		reader2 := &MockReader{}
		mockOpener.EXPECT().Open(ctx, k2).Return(reader2, nil).Times(1)
		mockCompiler.EXPECT().Compile(ctx, reader2).Return(v2, nil).Times(1)
		_, err = repo.Get(ctx, k2)
		require.NoError(t, err)

		// Verify snapshot is immutable
		val, ok := registry.Get(k2)
		assert.False(t, ok, "snapshot should not contain item added after snapshot")
		assert.Equal(t, "", val)

		// Verify original item is still there
		val1, ok1 := registry.Get(k1)
		assert.True(t, ok1)
		assert.Equal(t, v1, val1)
	})
}
