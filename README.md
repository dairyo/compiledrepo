# compiledrepo

`compiledrepo` is a Go library for managing cached repositories of compiled resources. It abstracts resource retrieval and transformation via `Opener` and `Compiler` interfaces, integrating caching and request coalescing.

## Getting Started

### Installation

```bash
go get github.com/dairyo/compiledrepo
```

### Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dairyo/compiledrepo"
	"github.com/dairyo/compiledrepo/driver/file"
	"github.com/dairyo/compiledrepo/driver/jsonschema"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func main() {
	ctx := context.Background()

	// Initialize driver implementations
	opener := file.NewOpener()
	compiler := jsonschema.NewCompiler[*os.File]()

	// Create a repository for JSON Schemas stored as files
	repo := compiledrepo.NewRepository[string, *os.File, *jsonschema.Schema](opener, compiler)

	// Retrieve a compiled schema
	schema, err := repo.Get(ctx, "schema.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Compiled schema: %v\n", schema)
}
```

## License

This project is licensed under the terms found in the [LICENSE](LICENSE) file.
