# compiledrepo — 理想形 API 仕様書
（ジェネリクス版・Go 的命名・簡潔エラー名）

以下のルールで記述する：

- コードブロックにはシグネチャのみ
- 実装仕様はコード外に文章で説明
- パッケージ名 × 型名で意味が決まる Go 的命名
- エラー名は簡潔に（Failed を付けない）
- ディレクトリ構造も含める

---

# 1. ディレクトリ構造

```
compiledrepo/
├── compiledrepo.go        // 抽象（Opener / Compiler / Iterator）
├── repository.go          // Repository（Get / Preload / Snapshot）
├── registry.go            // Registry（不変マップ）
├── errors.go              // エラー定義（ErrOpen / ErrCompile / ErrIterator）
│
├── open/
│   ├── file/
│   │   └── opener.go      // file.Opener
│   ├── bytesource/
│   │   └── opener.go      // bytesource.Opener
│   └── stringsource/
│       └── opener.go      // stringsource.Opener
│
├── compile/
│   ├── jsonschema/
│   │   └── compiler.go    // jsonschema.Compiler
│   └── adapters/
│       ├── bytes/
│       │   └── compiler.go   // bytes.Compiler
│       └── string/
│           └── compiler.go   // string.Compiler
│
└── iterate/
    └── file/
        └── iterator.go    // file.Iterator
```

---

# 2. Core Abstractions（抽象）

```go
type Opener[K comparable, R io.ReadCloser] interface {
    Open(ctx context.Context, key K) (R, error)
}

type Compiler[R io.ReadCloser, V any] interface {
    Compile(ctx context.Context, r R) (V, error)
}
```

### 仕様（説明）

- Opener は **K → R** を定義する
- Compiler は **R → V** を定義する
- R が一致しているため、Opener と Compiler は型レベルでペアになる
- R は io.ReadCloser 制約のみで、具体型は自由（*os.File, *bytes.Reader など）
- 詳細を隠さずに汎用性を維持できる

---

# 3. Repository

```go
type Repository[K comparable, R io.ReadCloser, V any] struct {
}

func NewRepository[K comparable, R io.ReadCloser, V any](
    opener Opener[K, R],
    compiler Compiler[R, V],
) *Repository[K, R, V]

func (r *Repository[K, R, V]) Get(ctx context.Context, key K) (V, error)

func (r *Repository[K, R, V]) Preload(ctx context.Context, it KeyIterator[K]) error

func (r *Repository[K, R, V]) Snapshot() Registry[K, V]
```

### 仕様（説明）

- Repository は Opener と Compiler のペアを保持
- R が一致しているため、型レベルで整合性が保証される
- Get の流れ：
  1. sync.Mapのキャッシュ確認
  2. singleflight による集約
  3. Open → Compile → キャッシュ保存
- panic は recover して error に変換
- Snapshot は不変マップを返す

---

# 4. Key Iterator

```go
type KeyIterator[K comparable] interface {
    All(ctx context.Context) iter.Seq2[K, error]
}
```

### 仕様（説明）

- キー列挙のストリームを返す
- Preload が利用する
- エラーはストリーム内で返される

---

# 5. Registry

```go
type Registry[K comparable, V any] struct {
}

func (reg Registry[K, V]) Get(key K) (V, bool)
```

### 仕様（説明）

- Snapshot により生成される不変マップ
- 読み取り専用で、Repository のキャッシュとは独立

---

# 6. Errors（簡潔なエラー定義）

```go
var (
    ErrOpen     error
    ErrCompile  error
    ErrIterator error
)
```

### 仕様（説明）

- ErrOpen
  - Opener が Reader を開けなかった場合
  - ラップ例：`fmt.Errorf("%w: %v", ErrOpen, err)`

- ErrCompile
  - Compiler が入力を処理できなかった場合

- ErrIterator
  - KeyIterator がキー列挙に失敗した場合

---

# 7. Concrete Implementations（具象）

## open/file

```go
package file

type Opener struct {
}

func (Opener) Open(ctx context.Context, path string) (*os.File, error)
```

### 仕様（説明）

- path をファイルパスとして扱い、*os.File を返す
- 型名は Opener で十分
- パッケージ名 file が役割を説明している

---

## open/bytesource

```go
package bytesource

type Opener[K comparable] struct {
}

func (Opener[K]) Open(ctx context.Context, key K) (*bytes.Reader, error)
```

### 仕様（説明）

- 任意のキー K から []byte を取得し、*bytes.Reader を返す
- Reader の具体型を隠さないため、Compiler が最適化可能

---

## open/stringsource

```go
package stringsource

type Opener[K comparable] struct {
}

func (Opener[K]) Open(ctx context.Context, key K) (*strings.Reader, error)
```

### 仕様（説明）

- 任意のキー K から string を取得し、*strings.Reader を返す

---

## compile/jsonschema

```go
package jsonschema

type Compiler struct {
}

func (Compiler) Compile(ctx context.Context, f *os.File) (*jsonschema.Schema, error)
```

### 仕様（説明）

- *os.File を読み取り、JSON Schema をコンパイル
- f.Seek() など具体型メソッドを利用可能

---

## compile/adapters/bytes

```go
package bytes

type Compiler[V any] struct {
}

func (Compiler[V]) Compile(ctx context.Context, r io.ReadCloser) (V, error)
```

### 仕様（説明）

- Reader を読み切って []byte に変換し、内部の CompileBytes に渡す

---

## compile/adapters/string

```go
package string

type Compiler[V any] struct {
}

func (Compiler[V]) Compile(ctx context.Context, r io.ReadCloser) (V, error)
```

### 仕様（説明）

- Reader を読み切って string に変換し、内部の CompileString に渡す

---

# 8. この API デザインの特徴（まとめ）

- パッケージ名 × 型名で意味が決まる Go 的命名
- Opener と Compiler は R を共有し、型レベルでペアリングされる
- R は io.ReadCloser 制約で汎用性を維持
- R の具体型は開示され、最適化や詳細利用が可能
- Repository はペアの整合性を型レベルで保証
- エラー名は簡潔で Go 的（Failed を付けない）
- 独自の硬直した型は導入しない
- ジェネリクスの本来の強みを最大限活かした設計
