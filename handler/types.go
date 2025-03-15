package handler

import "mime/multipart"

type errcode int

const (
	errBadRequest errcode = 10001 + iota
	errInternalServer
)

func (e errcode) String() string {
	switch e {
	case errBadRequest:
		return "请求内容有误"
	case errInternalServer:
		return "服务处理错误"
	default:
		return "位置错误"
	}
}

type importDataRequest struct {
	File     *multipart.FileHeader `form:"file" binding:"required"`
	ShipName string                `form:"shipName" binding:"required"`
	Cover    bool                  `form:"cover"`
}

type commonRequest struct {
	ShipName  string `form:"shipName" binding:"required"`
	StartDate string `form:"startDate" binding:"required,datetime=2006-01-02"`
	EndDate   string `form:"endDate" binding:"required,datetime=2006-01-02"`
}

type getOptimalShiftRequest struct {
	commonRequest
}

type getShiftPieRequest struct {
	commonRequest
}

type (
	getHistoryDataRequest struct {
		commonRequest
		getHistoryDataUri
	}
	getHistoryDataUri struct {
		ColumnName string `uri:"columnName" binding:"required"`
	}
)

type commonResponse struct {
	Code    errcode `json:"code"`
	Message string  `json:"message"`
	Data    any     `json:"data,omitempty"`
}

func success(data any) commonResponse {
	return commonResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func fail(code errcode, message string) commonResponse {
	return commonResponse{
		Code:    code,
		Message: message,
	}
}
