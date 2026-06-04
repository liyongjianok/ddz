package room

import "time"

const defaultTurnTimeout = 15 * time.Second

// TimeoutAction 表示超时系统代替玩家执行的动作类型。
type TimeoutAction string

const (
	TimeoutActionNone     TimeoutAction = ""
	TimeoutActionAutoBid  TimeoutAction = "auto_bid"
	TimeoutActionAutoPass TimeoutAction = "auto_pass"
	TimeoutActionAutoPlay TimeoutAction = "auto_play"
)
