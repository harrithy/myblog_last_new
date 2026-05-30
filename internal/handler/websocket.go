/**
 * WebSocket HTTP 处理器
 * 负责将 HTTP 请求升级为 WebSocket 连接，并集成 JWT 认证
 * 哼～这个处理器的认证逻辑可是和现有 JWT 中间件完美兼容的！主人你就放心用吧喵~
 */
package handler

import (
	"log"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	ws "myblog_last_new/pkg/websocket"
	"net/http"
)

// WebSocketHandler 处理 WebSocket 升级请求
type WebSocketHandler struct {
	userRepo *repository.UserRepository // 用于从 JWT 解析用户信息
}

// NewWebSocketHandler 创建 WebSocket 处理器
// 主人你看，这里用了 repository 依赖注入，和你项目里其他 Handler 风格完全统一喵~
func NewWebSocketHandler(userRepo *repository.UserRepository) *WebSocketHandler {
	return &WebSocketHandler{userRepo: userRepo}
}

// HandleConnection godoc
// @Summary WebSocket 连接端点
// @Description 将 HTTP 连接升级为 WebSocket 长连接，支持 JWT 认证（URL 参数 ?token=xxx 或 Authorization Header）。
// @Description 认证用户可订阅频道接收实时推送，匿名用户以访客身份连接。
// @Description 客户端连接后应发送 subscribe 消息订阅频道：
// @Description - {"type":"subscribe","channel":"global"} 全局通知
// @Description - {"type":"subscribe","channel":"comments:<id>"} 文章评论
// @Description - {"type":"subscribe","channel":"visits"} 实时访客
// @Tags websocket
// @Produce json
// @Param token query string false "JWT token（可选，用于认证连接）"
// @Success 101 "Switching Protocols - WebSocket 升级成功"
// @Router /ws [get]
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// 第一步：尝试从 URL 参数或 Header 中提取 JWT token 并解析用户身份
	var userID int
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// URL 参数没有？试试 Authorization Header（和 REST API 一致）
		tokenStr = extractBearerToken(r)
	}

	if tokenStr != "" {
		claims, err := middleware.ParseToken(tokenStr)
		if err != nil {
			log.Printf("[WS] Token 无效: %v，降级为匿名连接", err)
		} else {
			// JWT 中存的是 account，GetByLogin 支持 account/email 两种方式查找
			user, err := h.userRepo.GetByLogin(claims.Username)
			if err != nil {
				log.Printf("[WS] 查询用户失败: %v，降级为匿名连接", err)
			} else if user != nil {
				userID = user.ID
				log.Printf("[WS] 用户 %s (ID:%d) 通过 JWT 认证连接", claims.Username, userID)
			}
		}
	}

	if userID == 0 {
		log.Printf("[WS] 匿名访客连接")
	}

	// 第二步：升级到 WebSocket，注册到 Hub，启动读写循环
	if err := ws.Upgrade(w, r, userID); err != nil {
		response.InternalError(w, "WebSocket 升级失败: "+err.Error())
		return
	}
}

// GetOnlineCount godoc
// @Summary 查询 WebSocket 在线人数
// @Description 返回当前活跃的 WebSocket 连接数量，用于前端展示实时在线人数
// @Tags websocket
// @Produce json
// @Success 200 {object} response.APIResponse "在线人数统计"
// @Router /ws/online [get]
func (h *WebSocketHandler) GetOnlineCount(w http.ResponseWriter, r *http.Request) {
	count := ws.GetHub().OnlineCount()
	response.Success(w, map[string]int{"online_count": count})
}

// extractBearerToken 从 Authorization Header 中提取 Bearer token
// 格式: "Bearer <token>"
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}
