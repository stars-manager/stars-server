package constants

const (
	// HTTP 相关
	ContentTypeJSON = "application/json; charset=utf-8"
	ContentTypeText = "text/plain; charset=utf-8"

	// HTTP 头
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-ID"

	// HTTP 方法
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodOptions = "OPTIONS"

	// HTTP 状态码
	StatusOK                  = 200
	StatusCreated             = 201
	StatusNoContent           = 204
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusRequestEntityTooLarge = 413
	StatusInternalError       = 500
	StatusServiceUnavailable  = 503

	// 服务信息
	ServiceName = "hunyuan-api"

	// 限制
	MaxRequestBodySize = 1 << 20 // 1MB
	MaxHeaderSize      = 1 << 20 // 1MB

	// 超时
	DefaultReadTimeout  = 30  // 秒
	DefaultWriteTimeout = 300 // 秒 (5分钟)
)
