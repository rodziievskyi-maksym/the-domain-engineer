package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"eats/backend/common"
	"eats/backend/common/log"
)

func NewEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Server.WriteTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 5 * time.Second

	useMiddlewares(e)
	e.HTTPErrorHandler = HandleError
	e.Logger = common.NewEchoSlogAdapter(slog.Default())

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	return e
}

func HandleError(err error, c echo.Context) {
	log.FromContext(c.Request().Context()).With("error", err).Error("HTTP error")

	httpCode := http.StatusInternalServerError
	msg := any("Internal server error")

	httpErr := &echo.HTTPError{}
	if errors.As(err, &httpErr) {
		httpCode = httpErr.Code
		msg = httpErr.Message
	}

	jsonErr := c.JSON(
		httpCode,
		map[string]any{
			"error": msg,
		},
	)
	if jsonErr != nil {
		panic(err)
	}
}
