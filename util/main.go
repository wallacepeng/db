/*
  Copyright (c) 2012-2014 José Carlos Nieto, https://menteslibres.net/xiam

  Permission is hereby granted, free of charge, to any person obtaining
  a copy of this software and associated documentation files (the
  "Software"), to deal in the Software without restriction, including
  without limitation the rights to use, copy, modify, merge, publish,
  distribute, sublicense, and/or sell copies of the Software, and to
  permit persons to whom the Software is furnished to do so, subject to
  the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
  LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
  OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
  WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package util

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"menteslibres.net/gosexy/to"
	"upper.io/db"
)

var extRelationPattern = regexp.MustCompile(`\{(.+)\}`)
var columnCompareExclude = regexp.MustCompile(`[^a-zA-Z0-9]`)

var durationType = reflect.TypeOf(time.Duration(0))
var timeType = reflect.TypeOf(time.Time{})

type C struct {
	DB db.Database
}

type tagOptions map[string]bool

func parseTagOptions(s string) tagOptions {
	opts := make(tagOptions)
	chunks := strings.Split(s, ",")
	for _, chunk := range chunks {
		opts[strings.TrimSpace(chunk)] = true
	}
	return opts
}

// Based on http://golang.org/src/pkg/encoding/json/tags.go
func ParseTag(tag string) (string, tagOptions) {
	if i := strings.Index(tag, ","); i != -1 {
		return tag[:i], parseTagOptions(tag[i+1:])
	}
	return tag, parseTagOptions("")
}

func columnCompare(s string) string {
	return strings.ToLower(columnCompareExclude.ReplaceAllString(s, ""))
}

/*
	Returns the most appropriate struct field index for a given column name.

	If no column matches returns nil.
*/
func GetStructFieldIndex(t reflect.Type, columnName string) []int {

	n := t.NumField()

	for i := 0; i < n; i++ {

		field := t.Field(i)

		if field.PkgPath != "" {
			// Field is unexported.
			continue
		}

		// Attempt to use db:"column_name"
		fieldName, fieldOptions := ParseTag(field.Tag.Get("db"))

		// Deprecated "field" tag.
		if deprecatedField := field.Tag.Get("field"); deprecatedField != "" {
			fieldName = deprecatedField
		}

		// Deprecated "inline" tag.
		if deprecatedInline := field.Tag.Get("inline"); deprecatedInline != "" {
			fieldOptions["inline"] = true
		}

		// Matching fieldName
		if fieldName == "-" {
			continue
		}

		// Attempt to match field name.
		if fieldName == columnName {
			return []int{i}
		}

		if fieldName == "" {
			if columnCompare(field.Name) == columnCompare(columnName) {
				return []int{i}
			}
		}

		// Inline option.
		if fieldOptions["inline"] == true {
			index := GetStructFieldIndex(field.Type, columnName)
			if index != nil {
				res := append([]int{i}, index...)
				return res
			}
		}

	}

	// No match.
	return nil
}

/*
	Returns true if a table column looks like a struct field.
*/
func CompareColumnToField(s, c string) bool {
	return columnCompare(s) == columnCompare(c)
}

func ValidateSliceDestination(dst interface{}) error {

	var dstv reflect.Value
	var itemv reflect.Value
	var itemk reflect.Kind

	// Checking input
	dstv = reflect.ValueOf(dst)

	if dstv.IsNil() || dstv.Kind() != reflect.Ptr {
		return db.ErrExpectingPointer
	}

	if dstv.Elem().Kind() != reflect.Slice {
		return db.ErrExpectingSlicePointer
	}

	itemv = dstv.Elem()
	itemk = itemv.Type().Elem().Kind()

	if itemk != reflect.Struct && itemk != reflect.Map {
		return db.ErrExpectingSliceMapStruct
	}

	return nil
}

func StringToType(src string, dstt reflect.Type) (reflect.Value, error) {
	var srcv reflect.Value
	switch dstt {
	case durationType:
		srcv = reflect.ValueOf(to.Duration(src))
	case timeType:
		// Destination is time.Time
		srcv = reflect.ValueOf(to.Time(src))
	default:
		return StringToKind(src, dstt.Kind())
	}
	return srcv, nil
}

func StringToKind(src string, dstk reflect.Kind) (reflect.Value, error) {
	var srcv reflect.Value

	// Destination type.
	switch dstk {
	case reflect.Interface:
		// Destination is interface, nuff said.
		srcv = reflect.ValueOf(src)
	default:
		cv, err := to.Convert(src, dstk)
		if err != nil {
			return srcv, nil
		}
		srcv = reflect.ValueOf(cv)
	}

	return srcv, nil
}
