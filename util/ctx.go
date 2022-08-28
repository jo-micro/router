package util

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-micro.dev/v4/metadata"
)

// CtxFromRequest adds HTTP request headers to the context as metadata
func CtxFromRequest(c *gin.Context, r *http.Request) context.Context {
	md := make(metadata.Metadata, len(r.Header)+1)
	for k, v := range r.Header {
		// The space here is wanted spaces are not allowed in HTTP header fields.
		md["PROXY "+strings.Replace(k, " ", "_", -1)] = strings.Join(v, ",")
	}

	return metadata.MergeContext(c, md, true)
}
