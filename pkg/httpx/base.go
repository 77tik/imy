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
	// Code是业务状态码,不是HTTP响应码
	Code int `json:"code" xml:"code"`
	// Msg represents the business message, if Code = BusinessCodeOK,
	// and Msg is empty, then the Msg will be set to BusinessMsgOk.
	// Msg表示业务消息,如果Code是业务OK但是Msg是空的,那么Msg将被设置为ok
	Msg string `json:"msg" xml:"msg"`
	// 业务数据
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

// WrapBaseResponse 统一响应格式
func wrapBaseResponse(ctx context.Context, v any) BaseResponse[any] {
	var resp BaseResponse[any]
	switch data := v.(type) {
	case *errors.CodeMsg:
		// go-zero框架提供的错误类型
		resp.Code = data.Code
		resp.Msg = data.Msg
	case errors.CodeMsg:
		resp.Code = data.Code
		resp.Msg = data.Msg
	case *status.Status:
		// grpc状态类型，处理grpc调用返回的状态信息
		resp.Code = int(data.Code())
		resp.Msg = data.Message()
	case interface{ GRPCStatus() *status.Status }:
		// 处理实现了gRpc状态接口的类型
		resp.Code = int(data.GRPCStatus().Code())
		resp.Msg = data.GRPCStatus().Message()
	case BaseError:
		// 放Errcode了
		if err := data.GetErr(); err != nil {
			logc.Error(ctx, err)
		}
		resp.Code = data.GetCode()
		resp.Msg = data.GetMsg()
	case error:
		resp.Code = BusinessCodeError
		resp.Msg = data.Error()
	default:
		// 正常包装业务响应
		resp.Code = BusinessCodeOK
		resp.Msg = BusinessMsgOk
		resp.Data = v
	}

	return resp
}
