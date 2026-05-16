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

const defaultVisitContent = "standard visit record"

// VisitHandler handles visit-related requests.
type VisitHandler struct {
	visitRepo *repository.VisitRepository
	guestRepo *repository.GuestRepository
	ownerRepo *repository.OwnerVisitRepository
}

// NewVisitHandler creates a new VisitHandler.
func NewVisitHandler(visitRepo *repository.VisitRepository, guestRepo *repository.GuestRepository, ownerRepo *repository.OwnerVisitRepository) *VisitHandler {
	return &VisitHandler{
		visitRepo: visitRepo,
		guestRepo: guestRepo,
		ownerRepo: ownerRepo,
	}
}

// LogVisit godoc
// @Summary Create visit log
// @Description Records a user visit.
// @Tags visits
// @Accept json
// @Produce json
// @Param visit body models.VisitLog true "Visit payload"
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
		visit.Content = defaultVisitContent
	}

	id, err := h.visitRepo.Create(&visit)
	if err != nil {
		response.InternalError(w, "Failed to create visit log: "+err.Error())
		return
	}

	go h.visitRepo.CleanupOld(20)

	visit.ID = int(id)
	if visit.CreatedAt.IsZero() {
		visit.CreatedAt = models.CustomTime{Time: time.Now()}
	}

	response.Created(w, visit)
}

// GetVisitLogs godoc
// @Summary List visit logs
// @Description Returns paginated visit logs.
// @Tags visits
// @Produce json
// @Success 200 {object} response.APIResponse{data=[]models.VisitLog}
// @Router /visits [get]
func (h *VisitHandler) GetVisitLogs(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSizeValue := r.URL.Query().Get("pagesize")
	if pageSizeValue == "" {
		pageSizeValue = r.URL.Query().Get("pageSize")
	}
	if pageSizeValue == "" {
		pageSizeValue = r.URL.Query().Get("page_size")
	}

	pageSize, err := strconv.Atoi(pageSizeValue)
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
// @Summary Create guest record
// @Description Records a guest entry event.
// @Tags guest
// @Accept json
// @Produce json
// @Param record body models.GuestRecord true "Guest record payload"
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
// @Summary Get owner visit stats
// @Description Returns visit stats for the owner over the last N days.
// @Tags owner
// @Produce json
// @Param days query int false "Number of days, default 7"
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} response.APIResponse{data=object}
// @Security ApiKeyAuth
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
// @Summary Get owner today visits
// @Description Returns today's visit count for the owner.
// @Tags owner
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} response.APIResponse{data=object}
// @Security ApiKeyAuth
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
