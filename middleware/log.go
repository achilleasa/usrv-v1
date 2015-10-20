package middleware

import (
	"log"
	"time"

	"github.com/achilleasa/usrv"
)

func LogRequest(logger *log.Logger, handler usrv.Handler) usrv.Handler {
	return func(req usrv.Message, res usrv.Message) {
		defer func(start time.Time) {
			reqContent, _ := req.Content()
			resContent, err := res.Content()
			if err != nil {
				logger.Printf(
					"| %5s | %12d | from: %-20s | to: %-20s | %s |\n",
					"ERROR",
					time.Since(start).Nanoseconds(),
					req.From(),
					req.To(),
					len(reqContent),
					err.Error(),
				)
			} else {
				logger.Printf(
					"| %5s | %12d | from: %-20s | to: %-20s | reqLen: %5d | resLen: %5d |\n",
					"OK",
					time.Since(start).Nanoseconds(),
					req.From(),
					req.To(),
					len(reqContent),
					len(resContent),
				)
			}
		}(time.Now())

		handler(req, res)
	}
}
