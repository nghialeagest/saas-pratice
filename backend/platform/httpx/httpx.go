package httpx

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	platformerrors "saasbase/platform/errors"
)

// Registrar is implemented by modules that expose a public HTTP surface.
type Registrar interface {
	RegisterHTTP(r gin.IRouter)
}

type errorBody struct {
	Error string `json:"error"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Fail(c *gin.Context, err error) {
	var appErr *platformerrors.AppError
	if errors.As(err, &appErr) {
		status := appErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}

		if status >= 500 {
			log.Printf("[ERROR] %v", err)
			msg := appErr.Message
			if msg == "" {
				msg = "internal server error"
			}
			c.JSON(status, errorResponse{
				Code:    appErr.Code,
				Message: msg,
			})
			return
		}

		c.JSON(status, errorResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, errorResponse{
			Code:    "not_found",
			Message: "not found",
		})
	default:
		log.Printf("[ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    platformerrors.CodeInternal,
			Message: "internal server error",
		})
	}
}

func BadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, errorBody{Error: err.Error()})
}

