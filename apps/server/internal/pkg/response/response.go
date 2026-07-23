package response

import (
	"encoding/json"
	"net/http"
)

// Body 统一 API 响应信封，见 docs/04-api-spec.md。
type Body struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error 业务错误。
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Body{Success: true, Data: data})
}

func OKMessage(w http.ResponseWriter, data any, message string) {
	JSON(w, http.StatusOK, Body{Success: true, Data: data, Message: message})
}

func Fail(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, Body{
		Success: false,
		Error:   &Error{Code: code, Message: message},
	})
}

func BadRequest(w http.ResponseWriter, message string) {
	Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(w http.ResponseWriter, message string) {
	Fail(w, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(w http.ResponseWriter, message string) {
	Fail(w, http.StatusNotFound, "NOT_FOUND", message)
}

func Internal(w http.ResponseWriter, message string) {
	Fail(w, http.StatusInternalServerError, "INTERNAL", message)
}
