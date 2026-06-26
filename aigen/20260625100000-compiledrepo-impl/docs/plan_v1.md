# 計画書: compiledrepo 実装 (v1)

## 1. 実装目的
ジェネリクスを活用した、柔軟で型安全なリソース読み込み・コンパイル・キャッシュ機構 `compiledrepo` を実装する。Opener (K -> R) と Compiler (R -> V) を組み合わせ、Repository を通じてリソース V を効率的に取得・管理することを目的とする。

## 2. 実装チェックリスト

### Level 0: 基盤定義 (依存関係なし)
- [x] Sub-1: `errors.go` におけるセンチネルエラー (`ErrOpen`, `ErrCompile`, `ErrIterator`) の定義 (依存: なし) [4e07451]
- [x] Sub-2: `compiledrepo.go` における `Opener`, `Compiler` および `KeyIterator` インターフェースの定義 (依存: なし) [777fcdc]
- [x] Sub-3: `registry.go` における `Registry` 構造体および `Get` メソッドの実装 (依存: なし) [1285bb7]

### Level 1: 具象実装 (独立したロジック)
- [x] Sub-4: `open/file/opener.go` における `Opener.Open` の実装とテスト (依存: Sub-1, Sub-2) [8994d23]
- [x] Sub-5: `open/bytesource/opener.go` における `Opener.Open` の実装とテスト (依存: Sub-1, Sub-2) [6f9b473]
- [x] Sub-6: `open/stringsource/opener.go` における `Opener.Open` の実装とテスト (依存: Sub-1, Sub-2) [f3cabe7]
- [ ] Sub-7: `compile/jsonschema/compiler.go` における `Compiler.Compile` の実装とテスト (依存: Sub-1, Sub-2)
- [ ] Sub-8: `compile/adapters/bytes/compiler.go` における `Compiler.Compile` の実装とテスト (依存: Sub-1, Sub-2)
- [ ] Sub-9: `compile/adapters/string/compiler.go` における `Compiler.Compile` の実装とテスト (依存: Sub-1, Sub-2)
- [x] Sub-10: `iterate/file/iterator.go` における `KeyIterator.All` の実装とテスト (依存: Sub-1, Sub-2) [8b47f23]

### Level 2: Repository 統合ロジック
- [x] Sub-11: `repository.go` における `Repository` 構造体および `NewRepository` の実装 (依存: Sub-2) [718f2f2]
- [x] Sub-12: `repository.go` における `Repository.Get` メソッドの実装とテスト [405ebcf] (依存: Sub-1, Sub-11)
- [x] Sub-13: `repository.go` における `Repository.Preload` メソッドの実装とテスト (7f32888) (依存: Sub-1, Sub-10, Sub-12)
- [ ] Sub-14: `repository.go` における `Repository.Snapshot` メソッドの実装とテスト (依存: Sub-3, Sub-11)

---

## 3. 詳細設計

### [Sub-1]: センチネルエラー定義
- **定義内容**:
    - `ErrOpen`: リソースのオープンに失敗した場合。
    - `ErrCompile`: リソースのコンパイル（変換）に失敗した場合。
    - `ErrIterator`: キーの列挙に失敗した場合。
- **設計意図**: 呼び出し側がエラーの種類を判別し、適切なラップやリトライ判定を行えるようにする。

### [Sub-2]: Core Abstractions
- **`Opener[K comparable, R io.ReadCloser]`**:
    - `Open(ctx context.Context, key K) (R, error)`
- **`Compiler[R io.ReadCloser, V any]`**:
    - `Compile(ctx context.Context, r R) (V, error) `
- **設計意図**: 読み込み(Open)と変換(Compile)を分離し、中間型 R を介して型レベルでペアリングさせる。また、リソースの列挙を `KeyIterator` で抽象化し、`Preload` 等のバッチ処理を可能にす る。

### [Sub-3]: `Registry.Get`
- **シグネチャ**: `func (reg Registry[K, V]) Get(key K) (V, bool)`
- **ロジックステップ**:
    1. 内部で保持している不変マップから `key` を検索し、値と存在フラグを返す。
- **テストケース**:
    - Success: 存在するキーで値を正しく取得できること。
    - Success: 存在しないキーで `false` が返ること。

### [Sub-4]: `file.Opener.Open`
- **シグネチャ**: `func (Opener) Open(ctx context.Context, path string) (*os.File, error)`
- **ロジックステップ**:
    1. `os.Open(path)` を呼び出す。
    2. エラーが発生した場合、`ErrOpen` でラップして返す。
- **テストケース**:
    - Success: 実在するファイルパスで `*os.File` が返ること。
    - Error: 不在のファイルパスで `ErrOpen` が返ること。

### [Sub-5]: `bytesource.Opener.Open`
- **シグネチャ**: `func (Opener[K]) Open(ctx context.Context, key K) (*bytes.Reader, error)`
- **ロジックステップ**:
    1. キー `K` に対応するバイトスライスを取得する（実装上のモック/ストアから）。
    2. `bytes.NewReader` を用いて `*bytes.Reader` を生成する。
    3. 取得失敗時は `ErrOpen` でラップして返す。
    4. `io.ReadCloser` インターフェースを満たすため、適切にラップして返却する。
- **テストケース**:
    - Success: 有効なキーで `*bytes.Reader` が返ること。
    - Error: 不正なキーで `ErrOpen` が返ること。

### [Sub-6]: `stringsource.Opener.Open`
- **シグネチャ**: `func (Opener[K]) Open(ctx context.Context, key K) (*strings.Reader, error)`
- **ロジックステップ**:
    1. キー `K` に対応する文字列を取得する。
    2. `strings.NewReader` を用いて `*strings.Reader` を生成する。
    3. 取得失敗時は `ErrOpen` でラップして返す。
    4. `io.ReadCloser` インターフェースを満たすため、適切にラップして返却する。
- **テストケース**:
    - Success: 有効なキーで `*strings.Reader` が返ること。
    - Error: 不正なキーで `ErrOpen` が返ること。

### [Sub-7]: `jsonschema.Compiler.Compile`
- **シグネチャ**: `func (Compiler) Compile(ctx context.Context, f *os.File) (*jsonschema.Schema, error)`
- **ロジックステップ**:
    1. `f` から内容を読み取る。
    2. JSON Schema としてパースし、コンパイルを行う。
    3. 失敗した場合は `ErrCompile` でラップして返す。
- **テストケース**:
    - Success: 有効な JSON Schema ファイルで `*os.File` が返ること。
    - Error: 不正なフォーマットのファイルで `ErrCompile` が返ること。

### [Sub-8]: `bytes.Compiler.Compile`
- **シグネチャ**: `func (Compiler[V]) Compile(ctx context.Context, r io.ReadCloser) (V, error)`
- **ロジックステップ**:
    1. `io.ReadAll(r)` で全てのデータを読み切る。
    2. 読み取った `[]byte` を内部のコンパイル関数に渡す。
    3. 失敗時は `ErrCompile` でラップして返す。
- **テストケース**:
    - Success: 正しい入力ストリームで期待する型 `V` が返ること。
    - Error: 読み取り失敗またはコンパイル失敗で `ErrCompile` が返ること。

### [Sub-9]: `string.Compiler.Compile`
- **シグネチャ**: `func (Compiler[V]) Compile(ctx context.Context, r io.ReadCloser) (V, error)`
- **ロジックステップ**:
    1. `io.ReadAll(r)` で読み取り、`string` に変換する。
    2. 変換後の文字列を内部のコンパイル関数に渡す。
    3. 失敗時は `ErrCompile` でラップして返す。
- **テストケース**:
    - Success: 正しい入力ストリームで期待する型 `V` が返ること。
    - Error: 読み取り失敗またはコンパイル失敗で `ErrCompile` が返ること。

### [Sub-10]: `file.KeyIterator.All`
- **シグネチャ**: `func (Iterator) All(ctx context.Context) iter.Seq2[K, error)`
- **ロジックステップ**:
    1. 指定ディレクトリ内のファイルを走査する。
    2. `iter.Seq2` 形式で `(key, error)` を順次 yield する。
    3. 走査中のエラーは `ErrIterator` でラップして yield する。
- **テストケース**:
    - Success: 存在するファイル群がすべて列挙されること。
    - Error: ディレクトリ読み取り失敗時に `ErrIterator` が返ること。

### [Sub-11]: `Repository` 構造体と初期化
- **シグネチャ**: `func NewRepository[K comparable, R io.ReadCloser, V any](opener Opener[K, R], compiler Compiler[R, V]) *Repository[K, R, V]`
- **ロジックステップ**:
    1. `Opener` と `Compiler` を保持する `Repository` インスタンスを生成し、返す。
    2. キャッシュ用の `sync.Map` と `singleflight.Group` を初期化する。

### [Sub-12]: `Repository.Get`
- **シグネチャ**: `func (r *Repository[K, R, V]) Get(ctx context.Context, key K) (V, error)`
- **ロジックステップ**:
    1. キャッシュ (`sync.Map`) を確認し、存在すれば即座に返す。
    2. `singleflight.Group.Do` を使用して、同一キーの重複リクエストを抑制する。
    3. singleflight 内部処理:
        a. `opener.Open(ctx, key)` を呼び出す。エラー時は `ErrOpen` でラップして返す。
        b. `compiler.Compile(ctx, reader)` を呼び出す。エラー時は `ErrCompile` でラップして返す。
        c. 成功した結果をキャッシュに保存する。
    4. 全体的な `panic` を `recover` し、エラーとして返す。
- **テストケース**:
    - Success: 初回アクセスで Open -> Compile が走り、値が返ること。
    - Success: 2回目以降のアクセスでキャッシュから値が返ること。
    - Success: 同時リクエスト時に `Opener.Open` が1回しか呼ばれないこと（singleflight 検証）。
    - Error: `Open` 失敗時に `ErrOpen` が伝播すること。
    - Error: `Compile` 失敗時に `ErrCompile` が返ること。
    - Error: 内部で panic 発生時に適切にエラーとして返ること。

### [Sub-13]: `Repository.Preload`
- **シグネチャ**: `func (r *Repository[K, R, V]) Preload(ctx context.Context, it KeyIterator[K]) error`
- **ロジックステップ**:
    1. `it.All(ctx)` を呼び出し、キーのストリームを開始する。
    2. ストリームから得られる各キーに対し、`r.Get(ctx, key)` を呼び出してキャッシュにロードする。
    3. ストリーム内でエラーが発生した場合、そのエラーを返却する。
- **テストケース**:
    - Success: Iterator が返す全キーが `Get` され、キャッシュに格納されること。
    - Reviewer's Note: `Preload` の実装において、`KeyIterator.All` から得られるキーの列挙者がもともとエラーを返す可能性があるため、`Preload` 自体もそのエラーを伝播させる必要がある。
- **テストケース**:
    - Error: Iterator が `ErrIterator` を返した場合に停止し、エラーを返すこと。

### [Sub-14]: `Repository.Snapshot`
- **シグネチャ**: `func (r *Repository[K, R, V]) Snapshot() Registry[K, V]`
- **ロジックステップ**:
    1. 現在のキャッシュ (`sync.Map`) を `Registry` に正しくコピーする。
    2. コピーしたマップを保持する `Registry` インスタンスを生成して返す。
- **テストケース**:
    - Success: `Snapshot` 後に `Repository` のキャッシュが更新されても、`Registry` の内容は変わらないこと（不変性の検証）。
