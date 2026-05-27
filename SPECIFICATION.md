以下の仕様書に従い、 compiledrepo を実装してください。

あなたはメインエージェントです。自身で実装やレビューを決して行わないでください。
GEMINI.mdを良く読み流れをはずれないようにしてください。
これは multi agents での実装のテストです。 agent 利用に関する問題が発生した場合、それを記録し、別の AI が改善の為に利用できるファイルを出力してください。


---
# 論理設計仕様書: compiledrepo (v0.0.2)

## 1. 目的（エージェントへの指示）
本モジュールは、外部ソースからのデータ取得（Load）と型変換（Compile）を伴うリソースを管理する、Go 1.23+ 向けの汎用ライブラリです。
エージェントは、以下の「ディレクトリ・ファイル構成」、および「データ構造とデータフロー」を厳密に満たす実装ソースコードおよびテストコードを自動生成してください。

---

## 2. ディレクトリ・ファイル構成 (Physical Layout)

1つのモジュール（go.mod）内で、コアロジックと外部依存（具象実装）をパッケージレベルで完全に分離します。暗黙的な自動プリロードなどのマジックは排除し、明示的なAPI構成とします。

* **compiledrepo/** (モジュールルート : package compiledrepo)
  * **go.mod** (Go 1.23+ 指定)
  * **compiledrepo.go** (コアの型定義・インターフェース定義（契約）)
  * **repository.go** (Repository の動的ロード・キャッシュ・Preloadロジック)
  * **registry.go** (Registry の静的参照ロジック)
  * **errors.go** (ErrNotFound などの共通エラー定義)
  * **loaders/** (外部ソース取得の具象実装パッケージ群)
    * **fileloader/** (package fileloader)
      * **fileloader.go** (os / filepath を使った Loader & Preloader 実装)
  * **compilers/** (型変換の具象実装パッケージ群)
    * **jsonschema/** (package jsonschema)
      * **jsonschema.go** (外部パーサーに依存する Compiler 実装)

---

## 3. 各ファイルの論理設計とデータ構造

### 3.1. compiledrepo.go (ルートパッケージ：型・インターフェース定義)
具象ロジックや外部依存を持たせない、純粋なドメイン定義ファイル。

#### インターフェース
* `Loader`: `Load(ctx context.Context, id string) ([]byte, error)`
* `Compiler[T any]`: `func([]byte) (T, error)` (型変換を行う純粋関数型)
* `Preloader`: `All(ctx context.Context) iter.Seq[string, error]` (Go 1.23 標準 Iterator を用いた走査)

#### 構造体定義
* `Repository[T any]` 構造体
  * `loader`: Loader
  * `compiler`: Compiler[T]
  * `cache`: sync.Map (スレッドセーフな内部キャッシュ)
  * `sfg`: singleflight.Group (重複ロード防止用の排他制御ハンドラ)
* `Registry[T any]` 構造体
  * `resources`: map[string]T (読み取り専用マップ。Snapshot時に固定)

### 3.2. repository.go (ルートパッケージ：Repositoryの振る舞い)
* `NewRepository[T any](loader Loader, compiler Compiler[T]) *Repository[T]`
  * 初期状態（空キャッシュ）の `*Repository[T]` を生成するコンストラクタ。内部での暗黙的な型アサーションや自動プリロードは行わない。
* `Get(ctx context.Context, id string) (T, error)`
  * キャッシュを最優先で検索。ミス時は Loader.Load -> Compiler を経て内部キャッシュ（sync.Map）に格納してから返す（Lazy Load）。並行リクエスト時の重複ロードを防ぐため、Go言語のベストプラクティスである準標準パッケージ `golang.org/x/sync/singleflight` を使用した排他制御を実装すること。
* `Preload(ctx context.Context, p Preloader) error`
  * 明示的な呼び出し専用メソッド。`p.All(ctx)` の Iterator を for-range で回し、取得した id に対して自身の `Get(ctx, id)` を順次実行してキャッシュを事前に充填する。途中でエラーが出れば即座に中断してエラーを返す（Contextの制御とエラーハンドリングを利用側に委ねるため）。
* `Snapshot() *Registry[T]`
  * sync.Map 内のデータを、新しく確保した `map[string]T` へすべてコピーし、それを内包した `*Registry[T]` を生成して返す。

### 3.3. registry.go (ルートパッケージ：Registryの振る舞い)
* `Get(id string) (T, error)`
  * 引数に `context.Context` は受け取らない（I/Oが発生しないことをシグネチャで保証）。
  * 内部マップを検索し、存在すれば値を返し、なければ ErrNotFound を返す。データの書き込み（ミューテーション）を行うメソッドは一切提供しない。
* **※イミュータビリティに関する注記**:
  > 現在のGo言語において、ジェネリクス（`T`）に渡される参照型（ポインタ、マップ、スライス等）の完全なイミュータビリティを保証する汎用的な標準実装や明確なベストプラクティスは存在しません。そのため、方針として「取得したオブジェクトの内部状態を変更してはならない」旨をGoDocにて運用上の規約（契約）として明記するアプローチを採用してください。

### 3.4. loaders/fileloader/fileloader.go (サブパッケージ)
* ルートの Loader および Preloader を満たす具象構造体。
* `Load`: 引数の id をファイルパスとして扱い、`os.ReadFile` でバイトデータを返す。
* `All`: 指定されたルートディレクトリを `filepath.WalkDir` 等で走査し、見つかったファイルの識別子を `iter.Seq[string, error]` として順次 yield する（Iteratorの実装）。この際、必ず `ctx.Done()` のシグナルを監視し、キャンセル発生時は即座にリソースを解放してループを抜ける実装にすること。

### 3.5. compilers/jsonschema/jsonschema.go (サブパッケージ)
* ルートの Compiler[T] 型に適合する関数を提供する具象パッケージ。
* サードパーティ製の JSON Schema Parser ライブラリへの依存をこのファイル内に閉じ込め、ルートパッケージを汚染させない。

---

## 4. エージェントへの引き継ぎプロンプト

### エージェントへの指示文
上記の論理設計仕様書（v0.0.2）に記載された「ディレクトリ・ファイル構成」および「データ構造とデータフロー」に厳密に従って、Go 1.23+ のコード（Generics および iter.Seq2 を使用）を自動生成してください。

以下の要件を必ず満たしてください：

1. Loader からの暗黙的な自動プリロードなどのマジックは実装せず、利用側が明示的に `Preload(ctx, preloader)` を呼び出す設計を維持すること。
2. 同時リクエスト時のキャッシュスタンピード対策として、`Repository[T]` の公開メソッド `Get` の内部で `golang.org/x/sync/singleflight` を使用すること。
3. `Preloader.All` のイテレータ実装では、必ず `ctx.Done()` によるキャンセル伝播を適切に処理すること。
4. 各パッケージのカプセル化（非公開フィールドの徹底）を守り、スレッドセーフで高効率なコードを出力すること。
5. 参照型のミューテーション防止については、`Registry.Get` のGoDocに規約として明記すること。
6. それぞれのファイルに対応するユニットテストコード（`*_test.go`）も Go 標準の testing パッケージを用いて作成すること。
