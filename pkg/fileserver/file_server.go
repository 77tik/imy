package fileserver

import (
	"net/http"
	"path/filepath"
	"slices"
	"sync"

	"github.com/zeromicro/go-zero/rest"
	"imy/internal/config"
)

var (
	svrs sync.Map // map[Dir]ApiPrefix
)

func RunOptions(conf []config.FileServer) (opts []rest.RunOption) {
	for svr := range slices.Values(conf) {
		svrs.Store(svr.Dir, svr.ApiPrefix)
		opts = append(opts, WithFileServerGzip(svr.ApiPrefix, http.Dir(svr.Dir)))
	}
	return
}

func GetDlPath(absolutePath string) (downloadPath string) {
	svrs.Range(func(key, value any) bool {
		relativePath, err := filepath.Rel(key.(string), absolutePath)
		if err != nil {
			return true
		}
		downloadPath = filepath.Join(value.(string), relativePath)
		return false
	})

	if downloadPath == "" {
		downloadPath = absolutePath
	}
	return
}
