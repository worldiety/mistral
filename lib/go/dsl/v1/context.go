// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
)

// ctxKey is a package-private context key.
type ctxKey string

const (
	ctxDB                 ctxKey = "mistral-db"
	ctxHttpRequest        ctxKey = "http.Request"
	ctxHttpResponseWriter ctxKey = "http.ResponseWriter"
)

const (
	contentType  = "Content-Type"
	mimeTypeXML  = "application/xml"
	mimeTypeJSON = "application/json"
)

type httpError struct {
	status int
	msg    string
	cause  error
}

func (e httpError) HTTPError() bool {
	return true
}

func (e httpError) Error() string {
	return e.msg
}

func (e httpError) Status() int {
	return e.status
}

func (e httpError) Unwrap() error {
	return e.cause
}

// WithHttpRequest annotates a new Context with the according value.
func WithHttpRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, ctxHttpRequest, r)
}

// WithHttpResponse annotates a new Context with the according value.
func WithHttpResponse(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, ctxHttpResponseWriter, w)
}

// WithDB annotates a new Context with the according value.
func WithDB(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, ctxDB, db)
}

func mustRequest(ctx context.Context) *http.Request {
	req := ctx.Value(ctxHttpRequest).(*http.Request)
	if req == nil {
		panic("context does not contain a http.Request")
	}

	return req
}

// The stubBuilder just prints some debugging. The actual execution environment will replace it with
// an actual implementation.
type stubBuilder struct {
}

func (s *stubBuilder) Parameter(f func() (in interface{}, out interface{})) ProcBuilder {
	in, out := f()
	if in == nil {
		panic("in parameter must not be nil")
	}

	if out == nil {
		panic("out parameter must not be nil")
	}

	fmt.Println("declared request parameter type:")
	fmt.Println(reflect.ValueOf(in).String())
	buf, err := json.MarshalIndent(in, " ", " ")
	if err != nil {
		panic(err)
	}

	fmt.Println("example input:")
	fmt.Println(string(buf))

	fmt.Println("declared response parameter type:")
	fmt.Println(reflect.ValueOf(in).String())
	buf, err = json.MarshalIndent(out, " ", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println("example output:")
	fmt.Println(string(buf))

	return s
}

func (s *stubBuilder) Start(eval Evaluator) {
	// stub does intentionally not execute
}

func sortedLangTags(m map[string]Translation) []string {
	r := make([]string, 0, len(m))
	for key := range m {
		r = append(r, key)
	}

	sort.Strings(r)

	return r
}
