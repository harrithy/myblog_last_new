package handler

import (
	"io"
	"mime/multipart"
	"myblog_last_new/internal/response"
	"net/http"
	"time"
)

const ImageHostURL = "https://image.harrio.xyz/upload"

// UploadHandler proxies image uploads to the image host.
type UploadHandler struct{}

// NewUploadHandler creates a new UploadHandler.
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
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
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.BadRequest(w, "Failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			return
		}
		_, _ = io.Copy(part, file)
	}()

	req, err := http.NewRequest(http.MethodPost, ImageHostURL, pr)
	if err != nil {
		response.InternalError(w, "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		response.InternalError(w, "Failed to upload to image host: "+err.Error())
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
