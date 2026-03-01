// Package pkgresponse is for wrapping response
package pkgresponse

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/milfan/mp-go-pkg/errors"
	"github.com/minio/minio-go/v7"
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
	HTTPSuccess(ctx *gin.Context, message string, data interface{}, meta *Meta, statusCode int)

	// It used to return http response when has error
	HTTPError(ctx *gin.Context, err error)

	// It used to build meta response
	BuildMeta(page int, perPage int, count int64) *Meta

	// It used to show file image by minio
	FileMinio(ctx *gin.Context, object *minio.Object, contentType string)
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
func (r *ginResponse) HTTPError(ctx *gin.Context, err error) {

	var requestID string

	_ = ctx.Error(err)

	// get request id from middleware
	getRequestID, _ := ctx.Get("X-Request-ID")
	if getRequestID != nil {
		requestID = getRequestID.(string)
	}

	respError := pkgerrors.NewError(pkgerrors.UnknownError, nil)
	cerr, ok := err.(*pkgerrors.Error)
	if ok {
		respError = cerr
	}

	// add meta error
	respError.Meta = &pkgerrors.MetaError{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		RequestID: &requestID,
	}

	resp, _ := json.Marshal(respError)

	ctx.Writer.Header().Set(ContentType, ContentTypeJSON)
	ctx.Writer.WriteHeader(respError.HTTPCode)
	ctx.Writer.Write(resp)
	ctx.Abort()
}

func (r *ginResponse) HTTPSuccess(ctx *gin.Context, message string, data interface{}, meta *Meta, statusCode int) {

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

	ctx.Writer.Header().Set(ContentType, ContentTypeJSON)
	ctx.Writer.WriteHeader(httpStatus)
	ctx.Writer.Write(resp)
}

func (r *ginResponse) FileMinio(ctx *gin.Context, object *minio.Object, contentType string) {

	ctx.Header("Content-Type", contentType)
	_, err := io.Copy(ctx.Writer, object)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
	}
}

func New(
	logger *logrus.Logger,
) GinResponse {
	return &ginResponse{
		logger: logger,
	}
}
