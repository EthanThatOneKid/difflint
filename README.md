# difflint!

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

### Exhaustive switch statement

In programming languages lacking a comprehensive match statement for enumerations, our only option is to verify whether the switch statement aligns with the enumerated type.

```ts
//LINT.IF

enum Thing {
  ONE = 1,
  TWO = 2,
}

//LINT.END :thing_enum

//LINT.IF :thing_enum

switch (thing) {
  case Thing.ONE: {
    return doThingOne();
  }

  case Thing.TWO: {
    return doThingTwo();
  }
}

//LINT.END
```

### Custom file extensions

```bash
git diff | difflint --ext_map="difflint.json"
```

#### `difflint.json`

```json
{
  "yaml": ["#LINT.?"]
}
```

## Development

Run the tool from source with the Go toolchain:

```bash
go run cli/main.go --help
```

---

Created with 💖 by [**@EthanThatOneKid**](https://etok.codes/)
