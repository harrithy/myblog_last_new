package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse 定义标准的 API 响应格式
type APIResponse struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Msg   string      `json:"msg"`
	Total int64       `json:"total,omitempty"`
	Page  int         `json:"page,omitempty"`
}

// JSON 发送指定状态码的 JSON 响应
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Success 发送成功响应
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{
		Code: 200,
		Data: data,
		Msg:  "success",
	})
}

// SuccessWithPage 发送带分页信息的成功响应
func SuccessWithPage(w http.ResponseWriter, data interface{}, total int64, page int) {
	JSON(w, http.StatusOK, APIResponse{
		Code:  200,
		Data:  data,
		Msg:   "success",
		Total: total,
		Page:  page,
	})
}

// Created 发送 201 创建成功响应
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, APIResponse{
		Code: 200,
		Data: data,
		Msg:  "success",
	})
}

// Error 发送错误响应
func Error(w http.ResponseWriter, statusCode int, code int, msg string) {
	JSON(w, statusCode, APIResponse{
		Code: code,
		Data: nil,
		Msg:  msg,
	})
}

// BadRequest 发送 400 请求错误响应
func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, 400, msg)
}

// Unauthorized 发送 401 未授权响应
func Unauthorized(w http.ResponseWriter, msg string) {
	Error(w, http.StatusUnauthorized, 401, msg)
}

// NotFound 发送 404 未找到响应
func NotFound(w http.ResponseWriter, msg string) {
	Error(w, http.StatusNotFound, 404, msg)
}

// InternalError 发送 500 服务器内部错误响应
func InternalError(w http.ResponseWriter, msg string) {
	Error(w, http.StatusInternalServerError, 500, msg)
}
