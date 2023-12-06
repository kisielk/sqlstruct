# sqlstruct

sqlstruct provides some convenience functions for using structs with go's database/sql package

Documentation can be found at http://godoc.org/github.com/kisielk/sqlstruct

## Goals of this fork

1. Use generics instead of `interface{}`: In my opinion, using generics improves readability and allows for additional functionality.
2. Keep Language Injections intact: Intellij IDEs offer language injections that, in this case, provide support for sql-queries if literals match sql query patterns. This was previously not possible, because for injecting columns dynamically with sqlstruct, a pattern like `fmt.Sprintf("SELECT %s FROM ...", sqlstruct.Columns(mystruct{}))` had to be used.
3. Improve the package for my use-cases: While I would love for someone else to find use in this package, one of its main goals is to allow for the removal of boilerplate and redundant code in my private project by integrating patterns I often deploy.

## Usage

This package allows linking a struct and its database-counterpart, which means that `SELECT`-queries automatically reflect changes made to the datastructure by injecting the required columns into the query.

This works by extracting the exported fields of a struct, converting their names and inserting them into the given query.

### Basics

```go
var db *sql.DB

type User struct {
	ID       int
	UserName string
}

func GetAllUsers() (users []User, err error) {
	rows, err := db.Query(fmt.Sprintf("SELECT %s FROM users", sqlstruct.Columns[User]()))

	// ...
	return
}

```

The resulting query is: `SELECT id, username FROM users`. The column names can be modified using tags:

```go
type User struct {
	ID       int
	UserName string `sql:"user_name"`
}
```

The query now is: `SELECT id, user_name FROM users`

### Advanced

I've added the methods `Query` and `QueryRow`. They aim to remove boilerplate-code and keep goland language injections intact by working around the `fmt.Sprintf`.

The function `GetAllUsers` can now be simplified as follows:

```go
func GetAllUsers() {
	sqlstruct.SetDatabase(db)

	users, err := sqlstruct.Query[User]("SELECT * FROM users")
	// ...
}
```

The `*` in the query is replaced by the structs' columns, just as using `%s` and `sqlstruct.Columns(...)` would. It can be changed by setting the exported variable `sqlstruct.QueryReplace` to any other character.

Query row does the same, except not a slice is returned, but a single object:

```go
func GetUserByID(id int) {
    sqlstruct.SetDatabase(db)

    user, err := sqlstruct.QueryRow[User]("SELECT * FROM users WHERE id = ?", id)
    // ...
}
```