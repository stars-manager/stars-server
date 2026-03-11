package response

import (
	"encoding/json"
	"net/http"
)

// Code 错误码类型
type Code int

const (
	// 成功
	CodeSuccess Code = 0

	// 通用错误 1xxx
	CodeBadRequest   Code = 1001
	CodeInvalidParam Code = 1002
	CodeInvalidJSON  Code = 1003

	// 服务错误 2xxx
	CodeInternalError  Code = 2001
	CodeLLMError       Code = 2002
	CodeSessionTooLong Code = 2006
)

// CodeMsg 错误码对应的消息
var CodeMsg = map[Code]string{
	CodeSuccess:        "成功",
	CodeBadRequest:     "请求格式错误",
	CodeInvalidParam:   "参数校验失败",
	CodeInvalidJSON:    "JSON解析失败",
	CodeInternalError:  "服务器内部错误",
	CodeLLMError:       "LLM调用失败",
	CodeSessionTooLong: "会话历史过长，请开启新会话",
}

// ServiceError 自定义错误类型
type ServiceError struct {
	Code    Code
	Message string
}

// Error 实现error接口
func (e *ServiceError) Error() string {
	return e.Message
}

// NewError 创建错误
func NewError(code Code, message ...string) *ServiceError {
	msg := code.Msg()
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &ServiceError{
		Code:    code,
		Message: msg,
	}
}

// Msg 获取错误码消息
func (c Code) Msg() string {
	if msg, ok := CodeMsg[c]; ok {
		return msg
	}
	return "未知错误"
}

// Response 统一响应结构
type Response struct {
	Code    Code `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success 成功响应
func Success(w http.ResponseWriter, data any) {
	respondJSON(w, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: CodeSuccess.Msg(),
		Data:    data,
	})
}

// Error 错误响应
func Error(w http.ResponseWriter, code Code, message ...string) {
	msg := code.Msg()
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	statusCode := http.StatusBadRequest
	if code >= 2000 {
		statusCode = http.StatusInternalServerError
	}

	respondJSON(w, statusCode, Response{
		Code:    code,
		Message: msg,
	})
}

// respondJSON 返回JSON响应
func respondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
