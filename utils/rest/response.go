package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http/httputil"
)

type ResponseResult struct {
	Code   int           `json:"code"`
	Error  ResponseError `json:"error"`
	Result Result        `json:"result"`
}

type ResponseError struct {
}

type Result struct {
}

func RequestPrint(ctx *gin.Context) {
	dump, err := httputil.DumpRequest(ctx.Request, true)
	logrus.Debugf("解析请求: %s; err: %v. \n", string(dump), err)

	ctx.Next()
}
