package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
)

type RequestBodyKey struct{}

const (
	MaxRequestBodySize = 1024 * 1024 * 50 // 50MB
)

func LimitReader(r io.Reader, n int64) io.Reader { return &LimitedReader{r, n} }

type LimitedReader struct {
	R io.Reader
	N int64
}

var ErrLimitedReaderExceeded = errors.New("limited reader exceeded")

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, ErrLimitedReaderExceeded
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

func SetRequestBody(req *http.Request, body []byte) {
	ctx := req.Context()
	bufCtx := context.WithValue(ctx, RequestBodyKey{}, body)
	*req = *req.WithContext(bufCtx)
}

func GetRequestBody(req *http.Request) ([]byte, error) {
	contentType := req.Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" ||
		strings.HasPrefix(contentType, "multipart/form-data") {
		return nil, nil
	}

	requestBody := req.Context().Value(RequestBodyKey{})
	if requestBody != nil {
		b, _ := requestBody.([]byte)
		return b, nil
	}
	var buf []byte
	var err error
	defer func() {
		req.Body.Close()
		if err == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(buf))
		}
	}()
	if req.ContentLength <= 0 ||
		strings.HasPrefix(contentType, "application/json") {
		buf, err = io.ReadAll(LimitReader(req.Body, MaxRequestBodySize))
		if err != nil {
			if errors.Is(err, ErrLimitedReaderExceeded) {
				return nil, fmt.Errorf("request body too large, max: %d", MaxRequestBodySize)
			}
			return nil, fmt.Errorf("request body read failed: %w", err)
		}
	} else {
		if req.ContentLength > MaxRequestBodySize {
			return nil, fmt.Errorf("request body too large: %d, max: %d", req.ContentLength, MaxRequestBodySize)
		}
		buf = make([]byte, req.ContentLength)
		_, err = io.ReadFull(req.Body, buf)
	}
	if err != nil {
		return nil, fmt.Errorf("request body read failed: %w", err)
	}
	SetRequestBody(req, buf)
	return buf, nil
}

func UnmarshalBodyReusable(req *http.Request, v any) error {
	requestBody, err := GetRequestBody(req)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(requestBody, &v)
}

func UnmarshalBody2Node(req *http.Request) (ast.Node, error) {
	requestBody, err := GetRequestBody(req)
	if err != nil {
		return ast.Node{}, err
	}
	return sonic.Get(requestBody)
}
