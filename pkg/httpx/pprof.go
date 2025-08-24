package httpx

import (
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterPPROF(server *rest.Server) {
	if value := os.Getenv("OPERATING_ENV"); value == "dev" {

		server.AddRoutes([]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: pprof.Index,
			},
			{
				Method:  http.MethodGet,
				Path:    "/cmdline",
				Handler: pprof.Cmdline,
			},
			{
				Method:  http.MethodGet,
				Path:    "/profile",
				Handler: pprof.Profile,
			},
			{
				Method:  http.MethodGet,
				Path:    "/symbol",
				Handler: pprof.Symbol,
			},
			{
				Method:  http.MethodGet,
				Path:    "/trace",
				Handler: pprof.Trace,
			},
		}, rest.WithPrefix("/api/debug/pprof/"))
	}
}
