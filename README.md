# difflint

[![Go Reference](https://pkg.go.dev/badge/github.com/ethanthatonekid/difflint.svg)](https://pkg.go.dev/github.com/ethanthatonekid/difflint)

🔍 Git-based linter tool that scans code changes for compliance with defined rules in source code.

## Installation

```bash
go get -u github.com/EthanThatOneKid/difflint
```

## Usage

```bash
difflint --help
```

### Single file

```py
# File: ./main.py

#LINT.IF

print("Edit me first!")

#LINT.END bar

#LINT.IF :bar

print("Edit me second!")

#LINT.END
```

### Multiple files

```py
# File ./foo.py

#LINT.IF

print("Edit me first!")

#LINT.END bar
```

```py
# File: ./main.py

#LINT.IF ./foo.py:bar

print("Edit me second!")

#LINT.END
```

## Development

Run the tool from source with the Go toolchain:

```bash
go run cli/main.go --help
```

---

Created with 💖 by [**@EthanThatOneKid**](https://etok.codes/)