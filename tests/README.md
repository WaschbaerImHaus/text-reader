# Tests

Unit-Tests befinden sich bei den jeweiligen Packages:

- `../src/renderer/markdown_test.go` – Tests für Markdown-Parsing (12 Tests)

## Tests ausführen

```bash
cd ../src
PKG_CONFIG_PATH=pkgconfig:$PKG_CONFIG_PATH CGO_ENABLED=1 go test ./... -v
```

Oder via Makefile (aus Projektroot):

```bash
make test
```
