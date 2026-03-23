package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse defines the standard API response format.
type APIResponse struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Msg   string      `json:"msg"`
	Total int64       `json:"total,omitempty"`
	Page  int         `json:"page,omitempty"`
}

// JSON sends a JSON response with the provided HTTP status code.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// Success sends a 200 response body.
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{
		Code: 200,
		Data: data,
		Msg:  "success",
	})
}

// SuccessWithPage sends a paginated success response.
func SuccessWithPage(w http.ResponseWriter, data interface{}, total int64, page int) {
	JSON(w, http.StatusOK, APIResponse{
		Code:  200,
		Data:  data,
		Msg:   "success",
		Total: total,
		Page:  page,
	})
}

// Created sends a 201 response body.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, APIResponse{
		Code: 201,
		Data: data,
		Msg:  "success",
	})
}

// Error sends a standard error response.
func Error(w http.ResponseWriter, statusCode int, code int, msg string) {
	JSON(w, statusCode, APIResponse{
		Code: code,
		Data: nil,
		Msg:  msg,
	})
}

// BadRequest sends a 400 bad request response.
func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, 400, msg)
}

// Unauthorized sends a 401 unauthorized response.
func Unauthorized(w http.ResponseWriter, msg string) {
	Error(w, http.StatusUnauthorized, 401, msg)
}

// Forbidden sends a 403 forbidden response.
func Forbidden(w http.ResponseWriter, msg string) {
	Error(w, http.StatusForbidden, 403, msg)
}

// NotFound sends a 404 not found response.
func NotFound(w http.ResponseWriter, msg string) {
	Error(w, http.StatusNotFound, 404, msg)
}

// MethodNotAllowed sends a 405 method not allowed response.
func MethodNotAllowed(w http.ResponseWriter, msg string) {
	Error(w, http.StatusMethodNotAllowed, 405, msg)
}

// InternalError sends a 500 internal server error response.
func InternalError(w http.ResponseWriter, msg string) {
	Error(w, http.StatusInternalServerError, 500, msg)
}
