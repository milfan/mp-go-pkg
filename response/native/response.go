// Package pkgresponse is for wrapping response
package pkgresponse

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	pkgerrors "github.com/milfan/mp-go-pkg/errors"
	"github.com/sirupsen/logrus"
)

type Meta struct {
	Page      int `json:"page,omitempty"`
	PerPage   int `json:"perPage,omitempty"`
	Total     int `json:"total"`
	TotalPage int `json:"totalPage,omitempty"`
}

type ResponseMessage struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type GinResponse interface {
	// It used to return http response when it's ok
	HTTPSuccess(w http.ResponseWriter, message string, data interface{}, meta *Meta, statusCode int)

	// It used to return http response when has error
	HTTPError(w http.ResponseWriter, err error)

	// It used to build meta response
	BuildMeta(page int, perPage int, count int64) *Meta
}

type ginResponse struct {
	logger *logrus.Logger
}

func (r *ginResponse) BuildMeta(page int, perPage int, count int64) *Meta {
	x := math.Ceil(float64(count) / float64(perPage))
	totalPage := int(x)
	return &Meta{
		Page:      page,
		PerPage:   perPage,
		Total:     int(count),
		TotalPage: totalPage,
	}
}

// HTTPError implements IResponse.
func (r *ginResponse) HTTPError(w http.ResponseWriter, err error) {

	respError := pkgerrors.NewError(pkgerrors.UnknownError, nil)
	cerr, ok := err.(*pkgerrors.Error)
	if ok {
		respError = cerr
	}

	// add meta error
	respError.Meta = &pkgerrors.MetaError{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	resp, _ := json.Marshal(respError)

	w.Header().Set(ContentType, ContentTypeJSON)
	w.WriteHeader(respError.HTTPCode)
	w.Write(resp)
}

func (r *ginResponse) HTTPSuccess(w http.ResponseWriter, message string, data interface{}, meta *Meta, statusCode int) {

	response := ResponseMessage{
		Message: message,
		Data:    data,
		Meta:    meta,
	}
	resp, _ := json.Marshal(response)

	httpStatus := http.StatusOK
	if statusCode > 0 {
		httpStatus = statusCode
	}

	w.Header().Set(ContentType, ContentTypeJSON)
	w.WriteHeader(httpStatus)
	w.Write(resp)
}

func New(
	logger *logrus.Logger,
) GinResponse {
	return &ginResponse{
		logger: logger,
	}
}
