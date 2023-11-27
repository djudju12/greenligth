package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/djudju12/greenlight/internal/util"
	"github.com/djudju12/greenlight/internal/validator"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	TestField1 string `json:"test1"`
	TestField2 string `json:"test2"`
	TestField3 int    `json:"test3"`
}

func TestReadJsonForErrors(t *testing.T) {
	app := &application{}
	testCases := []struct {
		name     string
		jsonBody string
	}{
		{
			name: "syntaxError - badly-formed JSON",
			jsonBody: `
			{
				"test1": "hello,
				"test2": "world",
			}`,
		},
		{
			name: "Unexpected EOF - badly-formed JSON",
			jsonBody: `
			{
				"test1": "hello",
				"test2": "world"
			`,
		},
		{
			name: "Type Error - JSON contains incorrect type",
			jsonBody: `
			{
				"test1": "hello",
				"test2": "world",
				"test3": "sailor"
			}`,
		},
		{
			name:     "EOF Error - Empty JSON in body",
			jsonBody: ``,
		},
		{
			name: "Unknow Field",
			jsonBody: `
			{
				"test1": "hello",
				"test2": "world",
				"test3": 0,
				"test99": "this is uknow"
			}`,
		},
		{
			name: "Max Bytes Error - JSON is to large",
			jsonBody: fmt.Sprintf(
				`{"test1": "%s",}`, util.RandomString(JsonMaxBytes)),
		},
		{
			name: "Badly Formed JSON - body contains more data than 1 JSON",
			jsonBody: `
			{
				"test1": "hello",
				"test2": "world",
				"test3": 0
			},
			{
				"test1": "hello",
				"test2": "world",
				"test3": 0
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			dst := testStruct{}
			body := strings.NewReader(tc.jsonBody)
			request := httptest.NewRequest(http.MethodGet, "/", body)

			// when
			err := app.readJSON(nil, request, &dst)

			// then
			require.Error(t, err)
			t.Logf("error %s", err.Error())
		})
	}
}

func TestReadCSSV(t *testing.T) {
	app := &application{}
	testCases := []struct {
		name         string
		values       string
		defaultValue []string
		check        func(t *testing.T, values []string)
	}{
		{
			name:         "Read CSV receives an input and return the list",
			values:       "hello,world",
			defaultValue: []string{""},
			check: func(t *testing.T, values []string) {
				require.ElementsMatch(t, []string{"hello", "world"}, values)
			},
		},
		{
			name:         "Read CSV receives NO input and return default value",
			values:       "",
			defaultValue: []string{"hello", "world"},
			check: func(t *testing.T, values []string) {
				require.ElementsMatch(t, []string{"hello", "world"}, values)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			key := "values"
			qs := make(url.Values)
			qs.Add(key, tc.values)

			// when
			values := app.readCSV(qs, key, tc.defaultValue)

			// then
			tc.check(t, values)
		})
	}
}

func TestReadInt(t *testing.T) {
	app := &application{}

	testCases := []struct {
		name         string
		value        string
		defaultValue int
		check        func(t *testing.T, value int, v *validator.Validator)
	}{
		{
			name:         "Read int receives a input and return the integer",
			value:        "10",
			defaultValue: 0,
			check: func(t *testing.T, value int, v *validator.Validator) {
				require.Equal(t, value, 10)
				require.True(t, v.Valid())
			},
		},
		{
			name:         "Read int receives an invalid input and return the default value",
			value:        "invalid",
			defaultValue: 10,
			check: func(t *testing.T, value int, v *validator.Validator) {
				require.Equal(t, value, 10)
				require.False(t, v.Valid())
				_, ok := v.Errors["value"]
				require.True(t, ok)
			},
		},
		{
			name:         "Read int receives no input and return the default value",
			value:        "",
			defaultValue: 10,
			check: func(t *testing.T, value int, v *validator.Validator) {
				require.Equal(t, value, 10)
				require.True(t, v.Valid())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			key := "value"
			v := validator.New()
			qs := make(url.Values)
			qs.Add(key, tc.value)

			// when
			value := app.readInt(qs, key, tc.defaultValue, v)

			// then
			tc.check(t, value, v)
		})
	}
}
