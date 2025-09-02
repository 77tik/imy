package fileserver

import (
	"context"
	"net/http"
	"testing"

	reqv3 "github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"ldhydropower/internal/config"
	"ldhydropower/internal/helper/reqx"
	"ldhydropower/internal/helper/syncx"
)

func TestRunOptions(t *testing.T) {
	dirs := []config.FileServer{
		{
			ApiPrefix: "/static",
			Dir:       "./testdata",
		},
		{
			ApiPrefix: "/static/myfile",
			Dir:       "./testdata/nested",
		},
	}

	configYaml := `
Name: foo
Port: 54321
`

	var cnf rest.RestConf
	require.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	svr, err := rest.NewServer(cnf, RunOptions(dirs)...)
	require.Nil(t, err)

	syncx.GoRecover(svr.Start)
	t.Cleanup(svr.Stop)

	tests := []struct {
		name            string
		requestPath     string
		expectedStatus  int
		expectedContent string
	}{
		{
			name:            "serve plain file",
			requestPath:     "/static/example.txt",
			expectedStatus:  http.StatusOK,
			expectedContent: "1",
		},
		{
			name:           "non-matching path",
			requestPath:    "/other/path",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "non-existent file",
			requestPath:    "/static/non-existent.txt",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "websocket request",
			requestPath:    "/ws",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:            "serve index.html",
			requestPath:     "/static/index.html",
			expectedStatus:  http.StatusOK,
			expectedContent: "hello",
		},
		{
			name:            "Serve index.html in a nested directory",
			requestPath:     "/static/nested/index.html",
			expectedStatus:  http.StatusOK,
			expectedContent: "helloo",
		},
		{
			name:            "serve index.html with another server",
			requestPath:     "/static/myfile/index.html",
			expectedStatus:  http.StatusOK,
			expectedContent: "helloo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 2 {
				result, err := reqv3.R().
					SetContext(context.Background()).
					Get(reqx.GetURL("localhost:54321", tt.requestPath))
				require.Nil(t, err)
				require.Equal(t, tt.expectedStatus, result.StatusCode)
				if tt.expectedStatus != http.StatusOK {
					return
				}
				assert.True(t, result.IsSuccessState())
				if len(tt.expectedContent) > 0 {
					assert.Equal(t, tt.expectedContent, result.String())
				}
			}
		})
	}
}
