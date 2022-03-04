package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/silverton-io/honeypot/pkg/config"
	"github.com/silverton-io/honeypot/pkg/response"
)

func timeoutResponse(c *gin.Context) {
	log.Trace().Stack().Msg("request timed out")
	c.JSON(http.StatusRequestTimeout, response.Timeout)
}

func Timeout(conf config.Timeout) gin.HandlerFunc {
	// TODO: pass context down the line so events aren't passed to invalid if the request times out.
	return timeout.New(
		timeout.WithTimeout(time.Duration(conf.Ms)*time.Millisecond),
		timeout.WithHandler(func(c *gin.Context) {
			c.Next()
		}),
		timeout.WithResponse(timeoutResponse),
	)
}
