// Copyright 2012 Kamil Kisiel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package sqlstruct provides some convenience functions for using structs with
the Go standard library's database/sql package.

The package works with structs that are tagged with a "sql" tag that identifies
which column of a SQL query the field corresponds to.

For example:

	type T struct {
		F1 string `sql:"f1"`
		F2 string `sql:"f2"`
	}

	rows, err := db.Query(fmt.Sprintf("SELECT %s FROM tablename", sqlstruct.Columns(T)))
	...

	for rows.Next() {
		var t T
		err = sqlstruct.Scan(&t, rows)
		...
	}

	err = rows.Err() // get any errors encountered during iteration


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
type fieldInfo map[string]int

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

		// Skip unexported fields and those which are not tagged
		if f.PkgPath != "" || tag == "" {
			continue
		}

		finfo[tag] = i
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
		return err
	}

	for _, name := range cols {
		idx, ok := fieldInfo[name]
		var v interface{}
		if !ok {
			// There is no field mapped to this column so we discard it
			v = &sql.RawBytes{}
		} else {
			v = elem.Field(idx).Addr().Interface()
		}
		values = append(values, v)
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	return nil
}

// Columns returns a string containing a sorted, comma-separated list of column names as defined
// by the type s. s must be a struct that has exported fields tagged with the "sql" tag.
func Columns(s interface{}) string {
	v := reflect.ValueOf(s)
	fields := getFieldInfo(v.Type())

	names := make([]string, 0, len(fields))
	for f := range fields {
		names = append(names, f)
	}

	sort.Strings(names)
	return strings.Join(names, ", ")
}
