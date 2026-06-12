# Getting Started

The fastest path is the MVP usage guide:

- [Usage](usage.md)

If you just want the CLI:

```bash
go install github.com/gompdf/gompdf/cmd/gompdf@latest
gompdf input.html output.pdf
```

If you want to call GomPDF from Go:

```go
package main

import (
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	converter := gompdf.New()
	if err := converter.ConvertFile("input.html", "output.pdf"); err != nil {
		log.Fatal(err)
	}
}
```
