package handler

import (
	"io"
	"mime/multipart"
	"myblog_last_new/internal/response"
	"net/http"
)

const ImageHostURL = "https://image.harrio.xyz/upload"

// UploadHandler 处理文件上传代理
type UploadHandler struct{}

// NewUploadHandler 创建新的 UploadHandler
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// ProxyUpload godoc
// @Summary 代理上传图片到图床
// @Description 将图片上传到图床服务器并返回URL
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "图片文件"
// @Success 200 {array} object "上传成功返回图片信息"
// @Failure 400 {object} response.APIResponse "请求错误"
// @Failure 500 {object} response.APIResponse "上传失败"
// @Router /upload [post]
func (h *UploadHandler) ProxyUpload(w http.ResponseWriter, r *http.Request) {
	// 限制上传文件大小为 10MB
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	// 创建新的 multipart 请求
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			return
		}
		io.Copy(part, file)
	}()

	// 发送请求到图床
	req, err := http.NewRequest("POST", ImageHostURL, pr)
	if err != nil {
		response.InternalError(w, "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		response.InternalError(w, "Failed to upload to image host: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// 直接转发图床的响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
