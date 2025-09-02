package fileserver

import (
	"net/http"
	"unsafe"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"imy/pkg/fileserver/internal/fileserver"
)

type unsafeServer struct {
	_      unsafe.Pointer
	router httpx.Router
}

// WithFileServerGzip returns a RunOption to serve files (with gzip enabled) from given dir with given path.
func WithFileServerGzip(path string, fs http.FileSystem) rest.RunOption {
	return func(server *rest.Server) {
		userver := (*unsafeServer)(unsafe.Pointer(server))
		userver.router = newFileServingRouter(userver.router, path, fs)
	}
}

type fileServingRouter struct {
	httpx.Router
	middleware rest.Middleware
}

func newFileServingRouter(router httpx.Router, path string, fs http.FileSystem) httpx.Router {
	return &fileServingRouter{
		Router:     router,
		middleware: fileserver.Middleware(path, fs),
	}
}

func (f *fileServingRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.middleware(f.Router.ServeHTTP)(w, r)
}
