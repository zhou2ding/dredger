package handler

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

type apiResponse struct {
	Code    errcode `json:"code"`
	Message string  `json:"message"`
	Data    any     `json:"data,omitempty"`
}

func success(data any) apiResponse {
	return apiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func fail(code errcode, message string) apiResponse {
	return apiResponse{
		Code:    code,
		Message: message,
	}
}
