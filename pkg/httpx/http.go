package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
)

type RequestKey struct{}
type ResponseKey struct{}

// HttpInterceptor 拦截器，将请求和响应对象存储到上下文
func HttpInterceptor(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	ctx = context.WithValue(ctx, RequestKey{}, r)
	ctx = context.WithValue(ctx, ResponseKey{}, w)

	return ctx
}

// GetRequest 从上下文获取原始请求对象，逻辑层会用到
func GetRequest(ctx context.Context) (*http.Request, bool) {
	row := ctx.Value(RequestKey{})
	if r, ok := row.(*http.Request); ok {
		return r, true
	}
	return nil, false
}

// GetResponse 从上下文获取原始响应对象，逻辑层会用到
func GetResponse(ctx context.Context) (http.ResponseWriter, bool) {
	row := ctx.Value(ResponseKey{})
	if w, ok := row.(http.ResponseWriter); ok {
		return w, true
	}
	return nil, false
}

// CustomResponseWriter 自定义响应写入器，用于记录是否写入过数据
type CustomResponseWriter struct {
	http.ResponseWriter

	Wrote bool
}

// 如果JsonBaseResponse传入的是这个，那么就无法重复写入，防止logic层擅自写入，handler层不知道导致写了两次
// 因为http响应头只能设置一次，多次调用writeHeader会导致panic，一旦写入响应体，http状态码就会被永久锁定
func (c *CustomResponseWriter) Write(b []byte) (int, error) {
	n, err := c.ResponseWriter.Write(b)
	if err != nil {
		return n, err
	}

	c.Wrote = true
	return n, err
}

type FileType int

const (
	Pdf FileType = iota + 1
	Docx
	Excel
)

func SendFile(ctx context.Context, name string, modTime time.Time, content io.ReadSeeker, fileType FileType) {
	w, ok := GetResponse(ctx)
	if !ok {
		logx.Errorf("ctx not fund http.ResponseWriter")
		JsonBaseResponse(w, errors.New("file download error"))
		return
	}
	r, ok := GetRequest(ctx)
	if !ok {
		logx.Errorf("ctx not fund *http.Request")
		JsonBaseResponse(w, errors.New("file download error"))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(name)))

	switch fileType {
	case Pdf:
		w.Header().Set("Content-Type", "application/pdf")
	case Docx:
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case Excel:
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}

	http.ServeContent(w, r, name, modTime, content)
}

func SendFileCopy(ctx context.Context, name string, content io.Reader, fileType FileType) {
	w, ok := GetResponse(ctx)
	if !ok {
		logx.Errorf("ctx not fund http.ResponseWriter")
		JsonBaseResponse(w, errors.New("file download error"))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(name)))

	switch fileType {
	case Pdf:
		w.Header().Set("Content-Type", "application/pdf")
	case Docx:
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case Excel:
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}

	// 将文件内容复制到响应体中
	if _, err := io.Copy(w, content); err != nil {
		logc.Errorf(ctx, "SendFileCopyErr: %s", err)
		JsonBaseResponse(w, errors.New("file download error"))
		return
	}
}
