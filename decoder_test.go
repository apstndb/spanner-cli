package main

import (
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

func createRow(t *testing.T, values []interface{}) *spanner.Row {
	t.Helper()

	// column names are not important in this test, so use dummy name
	names := make([]string, len(values))
	for i := 0; i < len(names); i++ {
		names[i] = "dummy"
	}

	row, err := spanner.NewRow(names, values)
	if err != nil {
		t.Fatalf("Creating spanner row failed unexpectedly: %v", err)
	}
	return row
}

func equalStringSlice(a []string, b []string) bool {
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDecodeRow(t *testing.T) {
	validTests := []struct {
		Input    *spanner.Row
		Expected []string
	}{
		// basic type
		{createRow(t, []interface{}{true}), []string{"true"}},
		{createRow(t, []interface{}{[]byte{'a', 'b', 'c'}}), []string{"YWJj"}}, // base64 encode of 'abc'
		{createRow(t, []interface{}{1.23}), []string{"1.230000"}},
		{createRow(t, []interface{}{123}), []string{"123"}},
		{createRow(t, []interface{}{"foo"}), []string{"foo"}},
		{createRow(t, []interface{}{time.Unix(1516676400, 0)}), []string{"2018-01-23T03:00:00Z"}},
		{createRow(t, []interface{}{civil.DateOf(time.Unix(1516676400, 0))}), []string{"2018-01-23"}},

		// basic nullable type
		{createRow(t, []interface{}{spanner.NullBool{Bool: true, Valid: true}, spanner.NullBool{Bool: false, Valid: false}}), []string{"true", "NULL"}},
		{createRow(t, []interface{}{[]byte{'a', 'b', 'c'}, []byte(nil)}), []string{"YWJj", "NULL"}},
		{createRow(t, []interface{}{spanner.NullFloat64{Float64: 1.23, Valid: true}, spanner.NullFloat64{Float64: 0, Valid: false}}), []string{"1.230000", "NULL"}},
		{createRow(t, []interface{}{spanner.NullInt64{Int64: 123, Valid: true}, spanner.NullInt64{Int64: 0, Valid: false}}), []string{"123", "NULL"}},
		{createRow(t, []interface{}{spanner.NullString{StringVal: "foo", Valid: true}, spanner.NullString{StringVal: "", Valid: false}}), []string{"foo", "NULL"}},
		{createRow(t, []interface{}{spanner.NullTime{Time: time.Unix(1516676400, 0), Valid: true}, spanner.NullTime{Time: time.Unix(0, 0), Valid: false}}), []string{"2018-01-23T03:00:00Z", "NULL"}},
		{createRow(t, []interface{}{spanner.NullDate{Date: civil.DateOf(time.Unix(1516676400, 0)), Valid: true}, spanner.NullDate{Date: civil.DateOf(time.Unix(0, 0)), Valid: false}}), []string{"2018-01-23", "NULL"}},

		// array type
		{createRow(t, []interface{}{[]string{}}), []string{"[]"}},
		{createRow(t, []interface{}{[]bool{true, false}}), []string{"[true, false]"}},
		{createRow(t, []interface{}{[][]byte{{'a', 'b', 'c'}, []byte{'e', 'f', 'g'}}}), []string{"[YWJj, ZWZn]"}},
		{createRow(t, []interface{}{[]float64{1.23, 2.45}}), []string{"[1.230000, 2.450000]"}},
		{createRow(t, []interface{}{[]int64{123, 456}}), []string{"[123, 456]"}},
		{createRow(t, []interface{}{[]string{"foo", "bar"}}), []string{"[foo, bar]"}},
		{createRow(t, []interface{}{[]time.Time{time.Unix(1516676400, 0), time.Unix(1516680000, 0)}}), []string{"[2018-01-23T03:00:00Z, 2018-01-23T04:00:00Z]"}},
		{createRow(t, []interface{}{[]civil.Date{civil.DateOf(time.Unix(1516676400, 0)), civil.DateOf(time.Unix(1516762800, 0))}}), []string{"[2018-01-23, 2018-01-24]"}},

		// array nullable type
		{createRow(t, []interface{}{[]bool(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]byte(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]float64(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]int64(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]string(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]time.Time(nil)}), []string{"NULL"}},
		{createRow(t, []interface{}{[]civil.Date(nil)}), []string{"NULL"}},
	}

	for _, test := range validTests {
		got, err := DecodeRow(test.Input)
		if err != nil {
			t.Error(err)
		}

		if !equalStringSlice(got, test.Expected) {
			t.Errorf("DecodeRow(%q) = %v, but expected = %v", test.Input, got, test.Expected)
		}
	}
}
