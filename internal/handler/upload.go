package handler

import (
	"fmt"
	"io"
	"mime/multipart"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/response"
	"net/http"
	"slices"
	"strings"
	"time"
)

// UploadHandler proxies image uploads to the configured image host.
type UploadHandler struct {
	targetURL    string
	maxBytes     int64
	allowedTypes []string
	httpClient   *http.Client
}

// NewUploadHandler creates a new UploadHandler.
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{
		targetURL:    config.UploadProxyURL(),
		maxBytes:     config.UploadMaxBytes(),
		allowedTypes: config.UploadAllowedTypes(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProxyUpload godoc
// @Summary Proxy image upload
// @Description Upload an image to the remote image host and forward its response.
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file"
// @Success 200 {array} object
// @Failure 400 {object} response.APIResponse "Invalid request"
// @Failure 500 {object} response.APIResponse "Upload failed"
// @Security ApiKeyAuth
// @Router /upload [post]
func (h *UploadHandler) ProxyUpload(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.targetURL) == "" {
		response.InternalError(w, "Upload proxy target is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		response.BadRequest(w, "Failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if contentType == "" {
		sniffedType, err := sniffContentType(file)
		if err != nil {
			response.BadRequest(w, "Failed to detect file type: "+err.Error())
			return
		}
		contentType = sniffedType
	}

	if !h.isAllowedContentType(contentType) {
		response.BadRequest(w, "Unsupported file type: "+contentType)
		return
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("create multipart form file: %w", err))
			return
		}

		if _, err := io.Copy(part, file); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("copy uploaded file: %w", err))
			return
		}

		_ = pw.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, h.targetURL, pr)
	if err != nil {
		response.InternalError(w, "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := h.httpClient.Do(req)
	if err != nil {
		response.InternalError(w, "Failed to upload to image host: "+err.Error())
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		response.InternalError(w, "Failed to forward upload response: "+err.Error())
	}
}

func (h *UploadHandler) isAllowedContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return slices.Contains(h.allowedTypes, mediaType)
}

func sniffContentType(file multipart.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return http.DetectContentType(buffer[:n]), nil
}
