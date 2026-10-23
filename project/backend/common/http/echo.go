package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"eats/backend/common"
)

func NewEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Server.WriteTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 5 * time.Second

	useMiddlewares(e)
	e.HTTPErrorHandler = common.EchoErrorHandler
	e.Logger = common.NewEchoSlogAdapter(slog.Default())

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	return e
}
