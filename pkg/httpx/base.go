package httpx

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	"google.golang.org/grpc/status"
	// "google.golang.org/grpc/status"
)

var (
	// BusinessCodeOK represents the business code for success.
	BusinessCodeOK = 0
	// BusinessMsgOk represents the business message for success.
	BusinessMsgOk = "ok"

	BusinessCodeError = -1
)

// BaseResponse is the base response struct.
type BaseResponse[T any] struct {
	// Code represents the business code, not the http status code.
	Code int `json:"code" xml:"code"`
	// Msg represents the business message, if Code = BusinessCodeOK,
	// and Msg is empty, then the Msg will be set to BusinessMsgOk.
	Msg string `json:"msg" xml:"msg"`
	// Data represents the business data.
	Data T `json:"data,omitempty" xml:"data,omitempty"`
}

// JsonBaseResponse writes v into w with http.StatusOK.
func JsonBaseResponse(w http.ResponseWriter, v any) {
	httpx.OkJson(w, wrapBaseResponse(context.Background(), v))
}

// JsonBaseResponseCtx writes v into w with http.StatusOK.
func JsonBaseResponseCtx(ctx context.Context, w http.ResponseWriter, v any) {
	httpx.OkJsonCtx(ctx, w, wrapBaseResponse(ctx, v))
}

type BaseError interface {
	GetCode() int
	GetMsg() string
	GetErr() error
}

func wrapBaseResponse(ctx context.Context, v any) BaseResponse[any] {
	var resp BaseResponse[any]
	switch data := v.(type) {
	case *errors.CodeMsg:
		resp.Code = data.Code
		resp.Msg = data.Msg
	case errors.CodeMsg:
		resp.Code = data.Code
		resp.Msg = data.Msg
	case *status.Status:
		resp.Code = int(data.Code())
		resp.Msg = data.Message()
	case interface{ GRPCStatus() *status.Status }:
		resp.Code = int(data.GRPCStatus().Code())
		resp.Msg = data.GRPCStatus().Message()
	case BaseError:
		if err := data.GetErr(); err != nil {
			logc.Error(ctx, err)
		}
		resp.Code = data.GetCode()
		resp.Msg = data.GetMsg()
	case error:
		resp.Code = BusinessCodeError
		resp.Msg = data.Error()
	default:
		resp.Code = BusinessCodeOK
		resp.Msg = BusinessMsgOk
		resp.Data = v
	}

	return resp
}
