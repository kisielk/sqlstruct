# sqlstruct

sqlstruct provides some convenience functions for using structs with go's database/sql package

Documentation can be found at http://godoc.org/github.com/kisielk/sqlstruct

## Goals of this fork

1. Use generics instead of `interface{}`: In my opinion, using generics improves readability and allows for additional functionality.
2. Keep Language Injections intact: Intellij IDEs offer language injections that, in this case, provide support for sql-queries if literals match sql query patterns. This was previously not possible, because for injecting columns dynamically with sqlstruct, a pattern like `fmt.Sprintf("SELECT %s FROM ...", sqlstruct.Columns(mystruct{}))` had to be used.
3. Improve the package for my use-cases: While I would love for someone else to find use in this package, one of its main goals is to allow for the removal of boilerplate and redundant code in my private project by integrating patterns I often deploy.

## Usage

todo