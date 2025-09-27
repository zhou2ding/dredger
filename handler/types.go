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
	StartDate int64  `form:"startDate" binding:"required"`
	EndDate   int64  `form:"endDate" binding:"required"`
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

type setTheoryOptimalRequest struct {
	ShipName                     string  `json:"shipName" binding:"required"`
	Flow                         float64 `json:"flow" binding:"required"`
	Concentration                float64 `json:"concentration" binding:"required"`
	SPumpRpm                     float64 `json:"sPumpRpm" binding:"required"`
	CutterDepth                  float64 `json:"cutterDepth" binding:"required"`
	CarriageTravel               float64 `json:"carriageTravel" binding:"required"`
	HorizontalSpeed              float64 `json:"horizontalSpeed" binding:"required"`
	BoosterPumpDischargePressure float64 `json:"boosterPumpDischargePressure" binding:"required"`
	VacuumDegree                 float64 `json:"vacuumDegree" binding:"required"`
}

type getTheoryOptimalRequest struct {
	ShipName string `form:"shipName" binding:"required"`
}

type genSolidRequest struct {
	Action                  string  `json:"action"`
	GeologyDbFile           string  `json:"geologyDbFile"`
	CalculationBoundaryFile string  `json:"calculationBoundaryFile"`
	DesignDepthFile         string  `json:"designDepthFile"`
	MudlineFile             string  `json:"mudlineFile"`
	ReferenceZ              float64 `json:"referenceZ"`
	GridDistanceXY          float64 `json:"gridDistanceXY"`
	GridDistanceZ           float64 `json:"gridDistanceZ"`
	Threshold               float64 `json:"threshold"`
	PileX                   float64 `json:"pileX"`
	PileY                   float64 `json:"pileY"`
	ProfileX1               float64 `json:"profileX1"`
	ProfileY1               float64 `json:"profileY1"`
	ProfileX2               float64 `json:"profileX2"`
	ProfileY2               float64 `json:"profileY2"`
	SpecifiedX              float64 `json:"specifiedX"`
	SpecifiedY              float64 `json:"specifiedY"`
	SpecifiedLength         float64 `json:"specifiedLength"`
	SpecifiedWidth          float64 `json:"specifiedWidth"`
}

type serveReq struct {
	Path string `form:"path" json:"path" binding:"required"`
}

type OpenLocationRequest struct {
	Path string `json:"path" binding:"required"`
}

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
