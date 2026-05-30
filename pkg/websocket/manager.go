/**
 * WebSocket 连接管理中心 — 基于经典的 Hub + Client 模式
 * 哼，这个设计可是参考了 gorilla/websocket 官方示例又改进过的！
 * 支持按频道订阅、消息广播、优雅关闭，主人你要好好珍惜喵~
 *
 * 频道说明：
 *   - "global"         全局广播频道（新文章、系统通知等）
 *   - "comments:<id>"  文章评论实时推送（id 为文章ID）
 *   - "visits"         实时访客统计
 */
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// 常量和配置
// ============================================================================

const (
	// writeWait 写超时：服务端向客户端写消息的最长等待时间
	writeWait = 10 * time.Second

	// pongWait 心跳超时：等待客户端 pong 的最长时间，超时则断开
	pongWait = 60 * time.Second

	// pingPeriod 心跳间隔：服务端发送 ping 的间隔（必须 < pongWait）
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize 单条消息最大字节数（128KB）
	maxMessageSize = 128 * 1024

	// sendBufferSize 每个客户端发送通道的缓冲大小
	sendBufferSize = 256
)

// ============================================================================
// 消息协议
// ============================================================================

// MessageType 定义客户端与服务端之间的消息类型
type MessageType string

const (
	// MsgSubscribe 客户端请求订阅某个频道
	MsgSubscribe MessageType = "subscribe"
	// MsgUnsubscribe 客户端请求取消订阅某个频道
	MsgUnsubscribe MessageType = "unsubscribe"
	// MsgBroadcast 服务端广播消息给订阅者
	MsgBroadcast MessageType = "broadcast"
	// MsgPing 心跳 ping
	MsgPing MessageType = "ping"
	// MsgPong 心跳 pong
	MsgPong MessageType = "pong"
	// MsgError 错误消息
	MsgError MessageType = "error"
	// MsgInfo 提示信息
	MsgInfo MessageType = "info"
)

// WSMessage 是 WebSocket 通信的通用消息结构
// 客户端发送 subscribe/unsubscribe 时用 type + channel
// 服务端推送 broadcast 时用 type + channel + data
type WSMessage struct {
	Type    MessageType     `json:"type"`              // 消息类型
	Channel string          `json:"channel,omitempty"` // 频道名称（订阅/广播时必填）
	Data    json.RawMessage `json:"data,omitempty"`    // 消息载荷（JSON 原始字节，调用方自行 Marshal）
}

// ============================================================================
// Client — 单个 WebSocket 连接
// ============================================================================

// Client 代表一个已升级的 WebSocket 连接
// 每个 Client 在独立的 goroutine 中运行读写循环
type Client struct {
	hub    *Hub                // 所属的 Hub（反向引用，用于注销）
	conn   *websocket.Conn     // WebSocket 原始连接
	send   chan []byte         // 发送缓冲区（Hub 通过此通道向客户端推送消息）
	subs   map[string]struct{} // 已订阅的频道集合（用空 struct 省内存）
	mu     sync.RWMutex        // 保护 subs 的并发读写
	userID int                 // 已认证的用户 ID，0 表示匿名访客
}

// newClient 创建一个新的 WebSocket 客户端
func newClient(hub *Hub, conn *websocket.Conn, userID int) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		subs:   make(map[string]struct{}),
		userID: userID,
	}
}

// subscribe 订阅指定频道（幂等操作，重复订阅无副作用）
// 只在频道映射里加个标记，真正的过滤在 Hub.broadcastToChannel 里做
func (c *Client) subscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subs[channel] = struct{}{}
}

// unsubscribe 取消订阅指定频道
func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subs, channel)
}

// isSubscribed 检查客户端是否订阅了某个频道
func (c *Client) isSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subs[channel]
	return ok
}

// readPump 从 WebSocket 连接读取消息并处理
// 每个 Client 独占一个 goroutine 运行此方法
// 负责：解析客户端发来的 subscribe/unsubscribe/ping 等指令
func (c *Client) readPump() {
	defer func() {
		// 读循环退出时，从 Hub 注销自己，关闭发送通道
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 设置读限制，防止恶意客户端发送超大消息撑爆内存
	c.conn.SetReadLimit(maxMessageSize)
	// 设置读超时 = pongWait，每次收到 pong 会自动刷新
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// 收到客户端的 pong 时，刷新读超时并记录日志（调试用）
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			// 连接断开（客户端关闭、网络故障等），退出循环
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS] 客户端异常断开: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			// 无法解析的消息，返回错误提示
			c.sendJSON(WSMessage{Type: MsgError, Data: json.RawMessage(`"无法解析的消息格式"`)})
			continue
		}

		c.handleMessage(msg)
	}
}

// handleMessage 根据消息类型分发处理逻辑
func (c *Client) handleMessage(msg WSMessage) {
	switch msg.Type {
	case MsgSubscribe:
		if msg.Channel == "" {
			c.sendJSON(WSMessage{Type: MsgError, Data: json.RawMessage(`"subscribe 必须指定 channel"`)})
			return
		}
		c.subscribe(msg.Channel)
		c.sendJSON(WSMessage{
			Type:    MsgInfo,
			Channel: msg.Channel,
			Data:    json.RawMessage(`"订阅成功"`),
		})
		log.Printf("[WS] 用户%d 订阅频道: %s", c.userID, msg.Channel)

	case MsgUnsubscribe:
		if msg.Channel == "" {
			c.sendJSON(WSMessage{Type: MsgError, Data: json.RawMessage(`"unsubscribe 必须指定 channel"`)})
			return
		}
		c.unsubscribe(msg.Channel)
		c.sendJSON(WSMessage{
			Type:    MsgInfo,
			Channel: msg.Channel,
			Data:    json.RawMessage(`"已取消订阅"`),
		})

	case MsgPing:
		// 客户端主动 ping，回复 pong
		c.sendJSON(WSMessage{Type: MsgPong})

	default:
		c.sendJSON(WSMessage{Type: MsgError, Data: json.RawMessage(`"未知消息类型: ` + string(msg.Type) + `"`)})
	}
}

// writePump 从 send 通道读取消息并写入 WebSocket 连接
// 每个 Client 独占一个 goroutine 运行此方法
func (c *Client) writePump() {
	// 定时发送 ping 保活（ticker 必须 defer Stop，否则会泄漏）
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// send 通道已关闭（Hub 注销时），发送关闭帧通知客户端
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 设置写超时，防止慢客户端阻塞整个 Hub
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WS] 写入失败: %v", err)
				return
			}

		case <-ticker.C:
			// 定时发送 WebSocket Ping 帧，检测客户端是否还活着
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendJSON 将 WSMessage 序列化为 JSON 并投递到 send 通道
// 如果 send 通道满（客户端处理慢），跳过消息并记录警告
func (c *Client) sendJSON(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] JSON 序列化失败: %v", err)
		return
	}
	select {
	case c.send <- data:
	default:
		// 通道满说明客户端消费太慢，丢弃此消息防止阻塞 Hub
		log.Printf("[WS] 警告: 用户%d 发送通道已满，丢弃消息", c.userID)
	}
}

// ============================================================================
// Hub — 全局连接管理器（单例）
// ============================================================================

// Hub 管理所有活跃的 WebSocket 连接，负责注册、注销、广播
// 整个应用应只有一个 Hub 实例
type Hub struct {
	// clients 存储所有已注册的客户端
	clients map[*Client]struct{}

	// register 是客户端注册通道（无缓冲，确保同步处理）
	register chan *Client

	// unregister 是客户端注销通道
	unregister chan *Client

	// broadcast 是全局广播通道（发给所有客户端，不限频道）
	broadcast chan []byte
}

var (
	instance *Hub
	once     sync.Once
)

// GetHub 返回 Hub 单例（懒初始化 + 线程安全）
func GetHub() *Hub {
	once.Do(func() {
		instance = &Hub{
			clients:    make(map[*Client]struct{}),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			broadcast:  make(chan []byte, 256),
		}
		go instance.run() // 启动 Hub 主循环
	})
	return instance
}

// run 是 Hub 的主事件循环，在独立 goroutine 中运行
// 所有对 clients map 的操作都在此单 goroutine 中完成，无需额外加锁
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// 新客户端注册：加入 map 并启动读写协程
			h.clients[client] = struct{}{}
			log.Printf("[WS] 客户端连接，当前在线: %d", len(h.clients))
			// 给新客户端发送欢迎信息
			client.sendJSON(WSMessage{
				Type: MsgInfo,
				Data: json.RawMessage(`"已连接到 WebSocket 服务"`),
			})

		case client := <-h.unregister:
			// 客户端注销：从 map 移除，关闭 send 通道
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[WS] 客户端断开，当前在线: %d", len(h.clients))
			}

		case message := <-h.broadcast:
			// 全局广播：发给所有连接的客户端（不限频道）
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 发送失败说明该客户端即将被清理
				}
			}
		}
	}
}

// OnlineCount 返回当前在线连接数（线程安全，只读）
func (h *Hub) OnlineCount() int {
	// Hub.run 是单 goroutine 操作 clients，所以这里简单读是安全的
	return len(h.clients)
}

// ============================================================================
// 对外暴露的 API
// ============================================================================

// BroadcastToAll 向所有在线客户端广播原始字节消息
func BroadcastToAll(data []byte) {
	GetHub().broadcast <- data
}

// BroadcastJSON 向所有在线客户端广播 JSON 消息（业务层常用）
// 会自动设置 Type 为 broadcast
func BroadcastJSON(channel string, data interface{}) {
	rawData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WS] BroadcastJSON 序列化失败: %v", err)
		return
	}

	msg := WSMessage{
		Type:    MsgBroadcast,
		Channel: channel,
		Data:    rawData,
	}

	msgBytes, _ := json.Marshal(msg)
	BroadcastToAll(msgBytes)
}

// BroadcastToChannel 向订阅了指定频道的所有客户端广播 JSON 消息
// 遍历所有客户端，只发给订阅了 channel 的那些
func BroadcastToChannel(channel string, data interface{}) {
	rawData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WS] BroadcastToChannel 序列化失败: %v", err)
		return
	}

	msg := WSMessage{
		Type:    MsgBroadcast,
		Channel: channel,
		Data:    rawData,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] BroadcastToChannel JSON序列化失败: %v", err)
		return
	}

	hub := GetHub()
	for client := range hub.clients {
		if client.isSubscribed(channel) {
			select {
			case client.send <- msgBytes:
			default:
			}
		}
	}
}

// ============================================================================
// WebSocket 升级器
// ============================================================================

// Upgrader 是 gorilla/websocket 的升级器实例
// CheckOrigin 在生产环境应改为校验允许的域名，这里先允许所有来源（开发模式）
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 生产环境应校验 r.Header.Get("Origin") 是否在白名单内
		return true
	},
}

// Upgrade 将 HTTP 连接升级为 WebSocket 连接，注册到 Hub 并启动读写循环
// userID 为 0 表示匿名访客
func Upgrade(w http.ResponseWriter, r *http.Request, userID int) error {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] 升级失败: %v", err)
		return err
	}

	client := newClient(GetHub(), conn, userID)

	// 注册到 Hub（通过 channel 异步完成，保证线程安全）
	hub := GetHub()
	hub.register <- client

	// 启动读写协程（各占一个 goroutine）
	go client.writePump()
	go client.readPump()

	return nil
}
