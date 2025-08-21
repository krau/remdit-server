package config

import "time"

const (
	MaxFileSize = 1024 * 1024 * 2 // 2 MB
	
	// WebSocket心跳检测配置
	WSReadTimeout     = 60 * time.Second  // WebSocket读取超时时间
	WSPingInterval    = 15 * time.Second  // Ping发送间隔
	WSWriteTimeout    = 10 * time.Second  // WebSocket写入超时时间
	WSMaxPingFailures = 3                 // 最大连续ping失败次数
)
