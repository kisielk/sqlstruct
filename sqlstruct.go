// Copyright 2012 Kamil Kisiel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package sqlstruct provides some convenience functions for using structs with
the Go standard library's database/sql package.

The package matches struct field names to SQL query column names. A field can
also specify a matching column with "sql" tag, if it's different from field
name.  Unexported fields or fields marked with `sql:"-"` are ignored, just like
with "encoding/json" package.
Aliased tables in sql statement may be scanned into a specific structure identified
by the same alias, see the second example.

For example:

    type T struct {
        F1 string
        F2 string `sql:"field2"`
        F3 string `sql:"-"`
    }

    rows, err := db.Query(fmt.Sprintf("SELECT %s FROM tablename", sqlstruct.Columns(T)))
    ...

    for rows.Next() {
        err = sqlstruct.Scan(&t, rows)
        ...
    }

    err = rows.Err() // get any errors encountered during iteration

Example with aliased structures:

    type User struct {
        Id int `sql:"id"`
        Username string `sql:"username"`
        Email string `sql:"address"`
        Name string `sql:"name"`
        HomeAddress *Address `sql:"-"`
    }

    type Address struct {
        Id int `sql:"id"`
        City string `sql:"city"`
        Street string `sql:"address"`
    }

    ...

    user := new(User)
    address := new(Address)
    sql := `
SELECT %s, %s FROM users AS u
INNER JOIN address AS a ON a.id = u.address_id
WHERE u.username = ?
`
    sql = fmt.Sprintf(sql, sqlstruct.ColumnsAliased(*user, "u"), sqlstruct.ColumnsAliased(*address, "a"))
    rows, err := db.Query(sql, "gedi")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    if rows.Next() {
        err = sqlstruct.ScanAliased(user, rows, "u")
        if err != nil {
            log.Fatal(err)
        }
        err = sqlstruct.ScanAliased(address, rows, "a")
        if err != nil {
            log.Fatal(err)
        }
        user.HomeAddress = address
    }
    fmt.Printf("%+v", *user)
    // output: "{Id:1 Username:gedi Email:gediminas.morkevicius@gmail.com Name:Gedas HomeAddress:0xc21001f570}"
    fmt.Printf("%+v", *user.HomeAddress)
    // output: "{Id:2 City:Vilnius Street:Plento 34}"

*/
package sqlstruct

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// A cache of fieldInfos to save reflecting every time. Inspried by encoding/xml
var finfos map[reflect.Type]fieldInfo
var finfoLock sync.RWMutex

// tagName is the name of the tag to use on struct fields
const tagName = "sql"

// fieldInfo is a mapping of field tag values to their indices
type fieldInfo map[string][]int

func init() {
	finfos = make(map[reflect.Type]fieldInfo)
}

// Rows defines the interface of types that are scannable with the Scan function.
// It is implemented by the sql.Rows type from the standard library
type Rows interface {
	Scan(...interface{}) error
	Columns() ([]string, error)
}

// getFieldInfo creates a fieldInfo for the provided type. Fields that are not tagged
// with the "sql" tag and unexported fields are not included.
func getFieldInfo(typ reflect.Type) fieldInfo {
	finfoLock.RLock()
	finfo, ok := finfos[typ]
	finfoLock.RUnlock()
	if ok {
		return finfo
	}

	finfo = make(fieldInfo)

	n := typ.NumField()
	for i := 0; i < n; i++ {
		f := typ.Field(i)
		tag := f.Tag.Get(tagName)

		// Skip unexported fields or fields marked with "-"
		if f.PkgPath != "" || tag == "-" {
			continue
		}

		// Handle embedded structs
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			for k, v := range getFieldInfo(f.Type) {
				finfo[k] = append([]int{i}, v...)
			}
			continue
		}

		// Use field name for untagged fields
		if tag == "" {
			tag = f.Name
		}
		tag = strings.ToLower(tag)

		finfo[tag] = []int{i}
	}

	finfoLock.Lock()
	finfos[typ] = finfo
	finfoLock.Unlock()

	return finfo
}

// Scan scans the next row from rows in to a struct pointed to by dest. The struct type
// should have exported fields tagged with the "sql" tag. Columns from row which are not
// mapped to any struct fields are ignored. Struct fields which have no matching column
// in the result set are left unchanged.
func Scan(dest interface{}, rows Rows) error {
	return doScan(dest, rows, "")
}

// ScanAliased scans the next row from "rows" and looks for all "dest" structure fields
// defined in struct by "sql" tags, which maches columns prefixed by "alias". See
// "ColumnsAliased" function, it generates a select clause which contains all aliased
// "dest" structure fields
func ScanAliased(dest interface{}, rows Rows, alias string) error {
	return doScan(dest, rows, alias)
}

// Columns returns a string containing a sorted, comma-separated list of column names as
// defined by the type s. s must be a struct that has exported fields tagged with the "sql" tag.
func Columns(s interface{}) string {
	return strings.Join(cols(s), ", ")
}

// Columns generates a sorted in ascending order select clause which consists of all "s"
// struct fields tagged by "sql". It also aliases all these column names with "a". Later
// it can be scanned into a struct by the same alias. See "ScanAliased"
func ColumnsAliased(s interface{}, a string) string {
	names := cols(s)
	aliased := make([]string, 0, len(names))
	for _, n := range names {
		aliased = append(aliased, a+"."+n+" AS "+a+"_"+n)
	}
	return strings.Join(aliased, ", ")
}

func cols(s interface{}) []string {
	v := reflect.ValueOf(s)
	fields := getFieldInfo(v.Type())

	names := make([]string, 0, len(fields))
	for f := range fields {
		names = append(names, f)
	}

	sort.Strings(names)
	return names
}

func doScan(dest interface{}, rows Rows, alias string) (err error) {
	destv := reflect.ValueOf(dest)
	typ := destv.Type()

	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("dest must be pointer to struct; got %T", destv))
	}
	fieldInfo := getFieldInfo(typ.Elem())

	elem := destv.Elem()
	var values []interface{}

	cols, err := rows.Columns()
	if err != nil {
		return
	}

	for _, name := range cols {
		if len(alias) > 0 {
			name = strings.Replace(name, alias+"_", "", 1)
		}
		idx, ok := fieldInfo[strings.ToLower(name)]
		var v interface{}
		if !ok {
			// There is no field mapped to this column so we discard it
			v = &sql.RawBytes{}
		} else {
			v = elem.FieldByIndex(idx).Addr().Interface()
		}
		values = append(values, v)
	}

	err = rows.Scan(values...)
	return
}
