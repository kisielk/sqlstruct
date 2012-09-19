// Copyright 2012 Kamil Kisiel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package sqlstruct

import (
	"reflect"
	"testing"
)

type testType struct {
	FieldA string `sql:"field_a"`
	FieldB string // Not used
	FieldC string `sql:"field_c"`
}

// testRows is a mock version of sql.Rows which can only scan strings
type testRows struct {
	columns []string
	values  []interface{}
}

func (r testRows) Scan(dest ...interface{}) error {
	for i := range r.values {
		v := reflect.ValueOf(dest[i])
		if v.Kind() != reflect.Ptr {
			panic("Not a pointer!")
		}

		switch dest[i].(type) {
		case *string:
			*(dest[i].(*string)) = r.values[i].(string)
		default:
			// Do nothing. We assume the tests only use strings here
		}
	}
	return nil
}

func (r testRows) Columns() ([]string, error) {
	return r.columns, nil
}

func (r *testRows) addValue(c string, v interface{}) {
	r.columns = append(r.columns, c)
	r.values = append(r.values, v)
}

func TestColumns(t *testing.T) {
	var v testType
	e := "field_a, field_c"
	c := Columns(v)

	if c != e {
		t.Errorf("expected %q got %q", e, c)
	}
}

func TestScan(t *testing.T) {
	rows := testRows{}
	rows.addValue("field_a", "a")
	rows.addValue("field_b", "b")
	rows.addValue("field_c", "c")

	e := testType{"a", "", "c"}

	var r testType
	err := Scan(&r, rows)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if r != e {
		t.Errorf("expected %q got %q", e, r)
	}
}
