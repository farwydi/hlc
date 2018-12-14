package main

import (
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

func main() {
	if err := fasthttp.ListenAndServe(":801", requestHandler); err != nil {
		logger.Error("Error in ListenAndServe", zap.Error(err))
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	ctx.SuccessString("", "1")
}
