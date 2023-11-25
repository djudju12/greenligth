package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djudju12/greenlight/internal/util"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	TestField1 string `json:"test1"`
	TestField2 string `json:"test2"`
	TestField3 int    `json:"test3"`
}

func TestReadJsonForErrors(t *testing.T) {
	app := &application{}
	testCase := []struct {
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

	for _, tc := range testCase {
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
