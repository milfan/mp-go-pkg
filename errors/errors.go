// Package pkgerrors is for wrapping error
package pkgerrors

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"gorm.io/gorm"
)

const (
	UnknownError       = "UNKNOWN_ERROR"
	DB_TIMEOUT         = "DATABASE_CONNECTION_ERROR_10000"
	DB_TRANSIENT_ERROR = "DATABASE_CONNECTION_ERROR_10001"
	DB_AUTH_ERROR      = "DATABASE_CONNECTION_ERROR_10002"
	DB_INTERNAL_ERROR  = "DATABASE_CONNECTION_ERROR_10003"
	DB_DATA_NOT_FOUND  = "DATABASE_CONNECTION_ERROR_10004"

	APP_NETWORK_ERROR = "APP_NETWORK_ERROR_20001"
)

type ErrorRequest map[string]string

type (
	Error struct {
		ErrorCode     string       `json:"code"`
		HTTPCode      int          `json:"httpCode"`
		ClientMessage string       `json:"message"`
		ErrTrace      error        `json:"-"`
		ErrRequest    ErrorRequest `json:"errorRequest,omitempty"`
		Meta          *MetaError   `json:"meta,omitempty"`
	}
	MetaError struct {
		Timestamp string  `json:"timestamp"`
		RequestID *string `json:"requestID,omitempty"`
	}
)

func (c Error) Error() string {
	return fmt.Sprintf("CommonError: %s.", c.ClientMessage)
}

func (c *Error) CustomClientMessage(message string) *Error {
	c.ClientMessage = message
	return c
}

func NewError(commonErrCode string, traceErrorMessage error) *Error {

	isDBErr, err := mapDatabaseError(traceErrorMessage)
	if isDBErr {
		return err
	}

	dictionaries := errorDicts.Errors[commonErrCode]

	// to check if error in not listed
	if dictionaries == nil {
		return &Error{
			ClientMessage: "Unknown error", // this unknown client message if the error not registered
			ErrorCode:     commonErrCode,
			HTTPCode:      http.StatusInternalServerError,
			ErrTrace:      traceErrorMessage,
		}
	}
	dictionaries.ErrTrace = traceErrorMessage

	return dictionaries
}

func NewErrorValidate(
	commonErrCode string,
	errMessage interface{},
) *Error {
	dictionaries := errorDicts.Errors[commonErrCode]

	// to check if error in not listed
	if dictionaries == nil {
		return &Error{
			ClientMessage: "Unknown error", // this unknown client message if the error not registered
			ErrorCode:     commonErrCode,
			HTTPCode:      http.StatusInternalServerError,
		}
	}
	if _err, ok := errMessage.(validation.Errors); ok {
		dictionaries.ErrRequest = buildValidationError(_err)
	}

	return dictionaries
}

func buildValidationError(err error) ErrorRequest {
	var errors ErrorRequest = map[string]string{}

	errValidate := strings.Split(err.Error(), ";")
	for _, err := range errValidate {
		errPerField := strings.Split(err, ":")
		if len(errPerField[0]) <= 1 {
			errors["error"] = errPerField[0]
		} else {
			errors[strings.TrimSpace(errPerField[0])] = strings.TrimSpace(errPerField[1])
		}
	}

	return errors
}

func mapDatabaseError(err error) (bool, *Error) {
	if err == nil {
		return false, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	dicts := databaseErrorDicts()

	var ne net.Error
	if errors.As(err, &ne) {
		e := dicts[APP_NETWORK_ERROR]
		return true, &e
	}

	var syscallErr *syscall.Errno
	if errors.As(err, &syscallErr) {
		e := databaseErrorDicts()[APP_NETWORK_ERROR]
		return true, &e
	}

	if errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, context.DeadlineExceeded) {
		e := databaseErrorDicts()[DB_TIMEOUT]
		return true, &e
	}

	return false, nil
}

func databaseErrorDicts() map[string]Error {
	return map[string]Error{
		DB_TIMEOUT: {
			ClientMessage: "Database request timeout",
			ErrorCode:     DB_TIMEOUT,
			HTTPCode:      http.StatusGatewayTimeout,
		},

		DB_TRANSIENT_ERROR: {
			ClientMessage: "Database service temporarily unavailable",
			ErrorCode:     DB_TRANSIENT_ERROR,
			HTTPCode:      http.StatusServiceUnavailable,
		},

		DB_AUTH_ERROR: {
			ClientMessage: "Database authentication error",
			ErrorCode:     DB_AUTH_ERROR,
			HTTPCode:      http.StatusInternalServerError,
		},

		DB_INTERNAL_ERROR: {
			ClientMessage: "Database internal service error",
			ErrorCode:     DB_INTERNAL_ERROR,
			HTTPCode:      http.StatusInternalServerError,
		},

		APP_NETWORK_ERROR: {
			ClientMessage: "Internal network error",
			ErrorCode:     APP_NETWORK_ERROR,
			HTTPCode:      http.StatusInternalServerError,
		},
	}
}
