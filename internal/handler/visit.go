package handler

import (
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"time"
)

// VisitHandler 处理访问相关请求
type VisitHandler struct {
	visitRepo *repository.VisitRepository
	guestRepo *repository.GuestRepository
	ownerRepo *repository.OwnerVisitRepository
}

// NewVisitHandler 创建新的 VisitHandler
func NewVisitHandler(visitRepo *repository.VisitRepository, guestRepo *repository.GuestRepository, ownerRepo *repository.OwnerVisitRepository) *VisitHandler {
	return &VisitHandler{
		visitRepo: visitRepo,
		guestRepo: guestRepo,
		ownerRepo: ownerRepo,
	}
}

// LogVisit godoc
// @Summary 记录用户访问
// @Description 记录一次新的用户访问
// @Tags visits
// @Accept  json
// @Produce  json
// @Param   visit   body    models.VisitLog   true  "访问信息"
// @Success 201 {object} response.APIResponse{data=models.VisitLog}
// @Router /visits [post]
func (h *VisitHandler) LogVisit(w http.ResponseWriter, r *http.Request) {
	var visit models.VisitLog
	if err := json.NewDecoder(r.Body).Decode(&visit); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if visit.VisitTime.IsZero() {
		response.BadRequest(w, "Visit time is required")
		return
	}

	if visit.Content == "" {
		visit.Content = "普通访问记录"
	}

	id, err := h.visitRepo.Create(&visit)
	if err != nil {
		response.InternalError(w, "Failed to create visit log: "+err.Error())
		return
	}

	// 异步清理旧记录
	go h.visitRepo.CleanupOld(20)

	visit.ID = int(id)
	if visit.CreatedAt.IsZero() {
		visit.CreatedAt = models.CustomTime{Time: time.Now()}
	}

	response.Created(w, visit)
}

// GetVisitLogs godoc
// @Summary 获取所有访问日志
// @Description 检索所有用户访问日志的列表
// @Tags visits
// @Produce  json
// @Success 200 {object} response.APIResponse{data=[]models.VisitLog}
// @Router /visits [get]
func (h *VisitHandler) GetVisitLogs(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pagesize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	visits, total, err := h.visitRepo.GetAll(page, pageSize)
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.SuccessWithPage(w, visits, total, page)
}

// LogGuestRecord godoc
// @Summary 记录访客进入信息
// @Description 记录访客进入网站的时间和内容信息
// @Tags guest
// @Accept  json
// @Produce  json
// @Param   record   body    models.GuestRecord   true  "访客记录信息"
// @Success 201 {object} response.APIResponse{data=models.GuestRecord}
// @Router /guest [post]
func (h *VisitHandler) LogGuestRecord(w http.ResponseWriter, r *http.Request) {
	var record models.GuestRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if record.EntryTime.IsZero() {
		response.BadRequest(w, "Entry time is required")
		return
	}

	if record.Content == "" {
		response.BadRequest(w, "Content is required")
		return
	}

	id, err := h.guestRepo.Create(&record)
	if err != nil {
		response.InternalError(w, "Failed to create guest record: "+err.Error())
		return
	}

	record.ID = int(id)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = models.CustomTime{Time: time.Now()}
	}

	response.Created(w, record)
}

// GetOwnerVisitStats godoc
// @Summary 获取博客主人访问统计
// @Description 获取博客主人指定天数内每天访问次数的统计信息
// @Tags owner
// @Produce  json
// @Param   days   query    int     false  "获取最近多少天的数据，默认7天"
// @Success 200 {object} response.APIResponse{data=object}
// @Router /owner/visits [get]
func (h *VisitHandler) GetOwnerVisitStats(w http.ResponseWriter, r *http.Request) {
	days := r.URL.Query().Get("days")
	if days == "" {
		days = "7"
	}

	visitStats, totalVisits, err := h.ownerRepo.GetStats(days)
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"visit_stats":  visitStats,
		"total_visits": totalVisits,
		"days":         days,
	})
}

// GetOwnerTodayVisits godoc
// @Summary 获取博客主人今日访问次数
// @Description 获取博客主人今天的访问次数统计
// @Tags owner
// @Produce  json
// @Success 200 {object} response.APIResponse{data=object}
// @Router /owner/today-visits [get]
func (h *VisitHandler) GetOwnerTodayVisits(w http.ResponseWriter, r *http.Request) {
	todayVisits, err := h.ownerRepo.GetTodayVisits()
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, map[string]interface{}{
		"date":         time.Now().Format("2006-01-02"),
		"today_visits": todayVisits,
	})
}
