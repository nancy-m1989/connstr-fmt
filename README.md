# connstr-fmt

A parser and pretty-printer for ADO.NET/ODBC-style connection strings —
the `Key=Value; Key2=Value2` format used by SQL Server, most ODBC drivers,
and plenty of home-grown config formats that copied it.

Connection strings in this format are usually built by hand or by string
concatenation, and the quoting rules are easy to get subtly wrong: a
password containing a semicolon, a stray unescaped quote, a key typed
twice. When that happens, most drivers fail at connect time with a
generic error and no indication of where in the string the problem is.
This package parses the format properly and, when it rejects input, says
exactly where and why.

## Format

- Pairs are separated by `;`. Whitespace around keys and values is
  trimmed.
- A value can be bare (`Server=localhost`) or wrapped in a matching pair
  of `'` or `"` quotes.
- Quote a value to include a literal `;` in it, or to preserve leading or
  trailing whitespace.
- To include the quote character itself inside a quoted value, double it:
  `Password="a""b"` is the value `a"b`.
- Keys are case-insensitive and must be unique.
- A few key aliases are recognized as the same setting: `Server` and `Data
  Source`, `Uid` and `User Id`. `Get` matches across an alias pair, and
  using both aliases for the same setting is a duplicate key error, just
  like repeating the same key twice.

## Library usage

```go
package main

import (
	"fmt"
	"log"

	connstr "github.com/nancy-m1989/connstr-fmt"
)

func main() {
	cs, err := connstr.Parse(`Server = localhost ; Database=app ; Password = "p@ss;w0rd" `)
	if err != nil {
		log.Fatal(err)
	}

	if db, ok := cs.Get("Database"); ok {
		fmt.Println("database:", db)
	}

	fmt.Println(cs.Format())
	// Server=localhost; Database=app; Password="p@ss;w0rd"
}
```

`Format` reorders nothing — pairs stay in the order they were parsed — it
just normalizes spacing and quotes values only where quoting actually
matters.

## Command line

```
go run ./cmd/connstrfmt config.conf
```

Reads a connection string from the given file, or from stdin if no file
is given, and prints the normalized form to stdout. On a parse error it
prints the error, with the source line and a caret under the exact
column, to stderr and exits with status 1.

## Errors

Every rejection carries a line, a column, and (where useful) a second
reference position, for example a duplicate key's original definition.
Given a file `bad.conf` containing:

```
Password="oops
```

```
$ go run ./cmd/connstrfmt bad.conf
line 1, column 10: quoted value is never closed
Password="oops
         ^
```

Other conditions the parser catches the same way: an empty key (`=foo`),
a key with no `=` (`Foo;Bar=1`), a key defined twice, and stray
characters after a closing quote (`Key="a"b;`).

## Status

Early stage. The grammar covers the common ADO.NET/ODBC shape but not
every provider-specific convention. Only a couple of key aliases are
recognized so far (see above); the `Provider=` prefix some connection
strings carry, and percent-encoded values in URI-style connection
strings, are still out of scope.
