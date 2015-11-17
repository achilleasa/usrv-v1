package middleware

import (
	"time"

	"github.com/achilleasa/usrv"
)

func LogRequest(logger usrv.Logger, handler usrv.Handler) usrv.Handler {
	return func(req usrv.Message, res usrv.Message) {
		defer func(start time.Time) {
			reqContent, _ := req.Content()
			resContent, err := res.Content()
			if err != nil {
				logger.Error(
					"Request failed",
					"error", err,
					"time", time.Since(start).Nanoseconds(),
					"from", req.From(),
					"to", req.To(),
					"correlationId", req.CorrelationId(),
					"req_len", len(reqContent),
				)
			} else {
				logger.Info(
					"Processed request",
					"time", time.Since(start).Nanoseconds(),
					"from", req.From(),
					"to", req.To(),
					"correlationId", req.CorrelationId(),
					"req_len", len(reqContent),
					"res_len", len(resContent),
				)
			}
		}(time.Now())

		handler(req, res)
	}
}
