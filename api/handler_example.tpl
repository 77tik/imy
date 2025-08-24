package {{.PkgName}}

import (
    "net/http"

    {{.ImportPackages}}

    xhttp "imy/pkg/httpx"
)

func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        {{if .HasRequest}}var req types.{{.RequestType}}
        if err := xhttp.Parse(r, &req); err != nil {
            xhttp.JsonBaseResponseCtx(r.Context(), w, err)
            return
        }
        {{end -}}
        cw := &xhttp.CustomResponseWriter{
            ResponseWriter: w,
            Wrote:          false,
        }
        ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

        l := {{.LogicName}}.New{{.LogicType}}(ctx, svcCtx)
        {{if .HasResp}}resp, {{end}}err := l.{{.Call}}({{if .HasRequest}}&req{{end}})
        if err != nil {
            if !cw.Wrote {
                xhttp.JsonBaseResponseCtx(r.Context(), w, err)
            }
        } else {
            if !cw.Wrote {
                {{if .HasResp}}xhttp.JsonBaseResponseCtx(r.Context(), w, resp){{else}}xhttp.JsonBaseResponseCtx(r.Context(), w, nil){{end}}
            }
        }
    }
}
