package compiledrepo

import (
	"context"
	"fmt"
	"io"
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
		assert.Contains(t, err.Error(), "panic recovered during Get: something went wrong")
	})
}
