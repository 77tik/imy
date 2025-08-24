package httpx

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/zeromicro/go-zero/core/mapping"
	"github.com/zeromicro/go-zero/core/validation"
	httpxx "github.com/zeromicro/go-zero/rest/httpx"
)

const (
	// ApplicationJson stands for application/json.
	ApplicationJson = "application/json"
	// ContentType is the header key for Content-Type.
	ContentType = "Content-Type"
	// JsonContentType is the content type for JSON.
	JsonContentType = "application/json; charset=utf-8"
)

var maxBodyLen int64 = 1024 << 20 // 1024MB

var validator atomic.Value

// Parse parses the request.
func Parse(r *http.Request, v any) error {
	kind := mapping.Deref(reflect.TypeOf(v)).Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		if err := httpxx.ParsePath(r, v); err != nil {
			return err
		}

		if err := httpxx.ParseForm(r, v); err != nil {
			return err
		}

		if err := httpxx.ParseHeaders(r, v); err != nil {
			return err
		}
	}

	if err := ParseJsonBody(r, v); err != nil {
		return err
	}

	if valid, ok := v.(validation.Validator); ok {
		return valid.Validate()
	} else if val := validator.Load(); val != nil {
		return val.(httpxx.Validator).Validate(r, v)
	}

	return nil
}

// ParseJsonBody parses the post request which contains json in body.
func ParseJsonBody(r *http.Request, v any) error {
	if withJsonBody(r) {
		reader := io.LimitReader(r.Body, maxBodyLen)
		return mapping.UnmarshalJsonReader(reader, v)
	}

	return mapping.UnmarshalJsonMap(nil, v)
}

func withJsonBody(r *http.Request) bool {
	return r.ContentLength > 0 && strings.Contains(r.Header.Get(ContentType), ApplicationJson)
}
