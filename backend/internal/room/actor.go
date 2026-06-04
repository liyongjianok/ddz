package room

import (
	"errors"
	"time"

	"ddz/backend/internal/game"
)

var ErrRoomClosed = errors.New("room closed")

type roomCommandType string

const (
	roomCommandJoin      roomCommandType = "join"
	roomCommandLeave     roomCommandType = "leave"
	roomCommandReady     roomCommandType = "ready"
	roomCommandBid       roomCommandType = "bid"
	roomCommandPlayCards roomCommandType = "play_cards"
	roomCommandPass      roomCommandType = "pass"
	roomCommandTimeout   roomCommandType = "timeout"
	roomCommandSnapshot  roomCommandType = "snapshot"
	roomCommandView      roomCommandType = "view"
)

type roomCommand struct {
	typ           roomCommandType
	userID        string
	preferredSeat *int
	ready         bool
	score         int
	cards         []game.Card
	response      chan roomCommandResult
}

type roomCommandResult struct {
	room      Room
	view      *RoomSnapshot
	seatIndex int
	started   bool
	action    TimeoutAction
	err       error
}

// RoomActor 负责串行处理单个房间内的所有命令，保证房间状态变更按顺序执行。
type RoomActor struct {
	room *Room
	rng  game.RNG
	cmds chan roomCommand
}

// NewRoomActor 创建房间 Actor，并立即启动命令处理循环。
func NewRoomActor(room *Room, rng game.RNG) *RoomActor {
	actor := &RoomActor{
		room: room,
		rng:  rng,
		cmds: make(chan roomCommand),
	}

	go actor.loop()
	return actor
}

// Join 在房间 Actor 队列中处理加入房间请求。
func (a *RoomActor) Join(userID string, preferredSeat *int) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:           roomCommandJoin,
		userID:        userID,
		preferredSeat: preferredSeat,
	})
}

// Leave 在房间 Actor 队列中处理离开房间请求。
func (a *RoomActor) Leave(userID string) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:    roomCommandLeave,
		userID: userID,
	})
}

// Ready 在房间 Actor 队列中处理准备状态变更。
func (a *RoomActor) Ready(userID string, ready bool) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:    roomCommandReady,
		userID: userID,
		ready:  ready,
	})
}

// PlaceBid 在房间 Actor 队列中处理玩家叫分。
func (a *RoomActor) PlaceBid(userID string, score int) (Room, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandBid,
		userID:   userID,
		score:    score,
		response: result,
	}
	reply := <-result
	return reply.room, reply.err
}

// PlayCards 在房间 Actor 队列中处理玩家出牌。
func (a *RoomActor) PlayCards(userID string, cards []game.Card) (Room, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandPlayCards,
		userID:   userID,
		cards:    append([]game.Card(nil), cards...),
		response: result,
	}
	reply := <-result
	return reply.room, reply.err
}

// Pass 在房间 Actor 队列中处理玩家不出。
func (a *RoomActor) Pass(userID string) (Room, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandPass,
		userID:   userID,
		response: result,
	}
	reply := <-result
	return reply.room, reply.err
}

// HandleTimeout 将超时检查送入同一条房间命令队列处理。
func (a *RoomActor) HandleTimeout() (Room, TimeoutAction, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandTimeout,
		response: result,
	}
	reply := <-result
	return reply.room, reply.action, reply.err
}

// Snapshot 返回房间当前只读快照。
func (a *RoomActor) Snapshot() (Room, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandSnapshot,
		response: result,
	}
	reply := <-result
	return reply.room, reply.err
}

// BuildSnapshot 生成指定玩家的专属房间快照。
func (a *RoomActor) BuildSnapshot(userID string) (*RoomSnapshot, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandView,
		userID:   userID,
		response: result,
	}
	reply := <-result
	return reply.view, reply.err
}

func (a *RoomActor) call(cmd roomCommand) (Room, int, bool, error) {
	result := make(chan roomCommandResult, 1)
	cmd.response = result
	a.cmds <- cmd
	reply := <-result
	return reply.room, reply.seatIndex, reply.started, reply.err
}

func (a *RoomActor) loop() {
	for cmd := range a.cmds {
		var result roomCommandResult
		switch cmd.typ {
		case roomCommandJoin:
			result.room, result.seatIndex, result.started, result.err = a.handleJoin(cmd.userID, cmd.preferredSeat)
		case roomCommandLeave:
			result.room, result.seatIndex, result.started, result.err = a.handleLeave(cmd.userID)
		case roomCommandReady:
			result.room, result.seatIndex, result.started, result.err = a.handleReady(cmd.userID, cmd.ready)
		case roomCommandBid:
			result.room, result.err = a.handlePlaceBid(cmd.userID, cmd.score)
		case roomCommandPlayCards:
			result.room, result.err = a.handlePlayCards(cmd.userID, cmd.cards)
		case roomCommandPass:
			result.room, result.err = a.handlePass(cmd.userID)
		case roomCommandTimeout:
			result.room, result.action, result.err = a.handleTimeout()
		case roomCommandSnapshot:
			result.room = a.room.Snapshot()
		case roomCommandView:
			result.view, result.err = BuildRoomSnapshot(a.room, cmd.userID)
		default:
			result.err = ErrInvalidRoomConfig
		}
		cmd.response <- result
	}
}

func (a *RoomActor) handleJoin(userID string, preferredSeat *int) (Room, int, bool, error) {
	if a.room.Status == RoomStatusClosed {
		return Room{}, -1, false, ErrRoomClosed
	}
	if a.room.Status != RoomStatusWaiting {
		return Room{}, -1, false, ErrGameAlreadyStarted
	}
	if len(a.room.Seats) >= a.room.MaxPlayers {
		return Room{}, -1, false, ErrRoomFull
	}

	for _, seat := range a.room.Seats {
		if seat.UserID == userID {
			return a.room.Snapshot(), seat.SeatIndex, a.room.CurrentGame != nil, nil
		}
	}

	seatIndex, err := firstAvailableSeat(a.room, preferredSeat)
	if err != nil {
		return Room{}, -1, false, err
	}

	now := time.Now().UTC()
	a.room.Seats = append(a.room.Seats, Seat{
		UserID:    userID,
		SeatIndex: seatIndex,
		Ready:     false,
		JoinedAt:  now,
	})
	a.room.UpdatedAt = now

	started, err := a.tryStartGame()
	if err != nil {
		return Room{}, -1, false, err
	}

	return a.room.Snapshot(), seatIndex, started, nil
}

func (a *RoomActor) handleLeave(userID string) (Room, int, bool, error) {
	if a.room.Status != RoomStatusWaiting {
		return Room{}, -1, false, ErrGameAlreadyStarted
	}

	seatIndex := -1
	removeIndex := -1
	for i, seat := range a.room.Seats {
		if seat.UserID == userID {
			seatIndex = seat.SeatIndex
			removeIndex = i
			break
		}
	}
	if removeIndex < 0 {
		return Room{}, -1, false, ErrUserNotInRoom
	}

	a.room.Seats = append(a.room.Seats[:removeIndex], a.room.Seats[removeIndex+1:]...)
	a.room.UpdatedAt = time.Now().UTC()
	if len(a.room.Seats) == 0 {
		a.room.Status = RoomStatusClosed
		a.room.DeadlineAt = time.Time{}
	}

	return a.room.Snapshot(), seatIndex, false, nil
}

func (a *RoomActor) handleReady(userID string, ready bool) (Room, int, bool, error) {
	if a.room.Status != RoomStatusWaiting {
		return Room{}, -1, false, ErrGameAlreadyStarted
	}

	seatIndex := -1
	for i := range a.room.Seats {
		if a.room.Seats[i].UserID == userID {
			seatIndex = a.room.Seats[i].SeatIndex
			a.room.Seats[i].Ready = ready
			a.room.UpdatedAt = time.Now().UTC()
			break
		}
	}
	if seatIndex < 0 {
		return Room{}, -1, false, ErrUserNotInRoom
	}

	started, err := a.tryStartGame()
	if err != nil {
		return Room{}, -1, false, err
	}

	return a.room.Snapshot(), seatIndex, started, nil
}

func (a *RoomActor) handlePlaceBid(userID string, score int) (Room, error) {
	currentGame := a.room.CurrentGame
	if currentGame == nil {
		return Room{}, ErrGameNotStarted
	}

	player, ok := findGamePlayerByUserID(currentGame.Players, userID)
	if !ok {
		return Room{}, ErrUserNotInRoom
	}

	if err := currentGame.PlaceBid(player.SeatIndex, score, a.rng); err != nil {
		return Room{}, err
	}
	if err := a.syncRoomGameState(); err != nil {
		return Room{}, err
	}

	return a.room.Snapshot(), nil
}

func (a *RoomActor) handlePlayCards(userID string, cards []game.Card) (Room, error) {
	currentGame := a.room.CurrentGame
	if currentGame == nil {
		return Room{}, ErrGameNotStarted
	}

	player, ok := findGamePlayerByUserID(currentGame.Players, userID)
	if !ok {
		return Room{}, ErrUserNotInRoom
	}

	if err := currentGame.PlayCards(player.SeatIndex, cards); err != nil {
		return Room{}, err
	}
	if err := a.syncRoomGameState(); err != nil {
		return Room{}, err
	}

	return a.room.Snapshot(), nil
}

func (a *RoomActor) handlePass(userID string) (Room, error) {
	currentGame := a.room.CurrentGame
	if currentGame == nil {
		return Room{}, ErrGameNotStarted
	}

	player, ok := findGamePlayerByUserID(currentGame.Players, userID)
	if !ok {
		return Room{}, ErrUserNotInRoom
	}

	if err := currentGame.Pass(player.SeatIndex); err != nil {
		return Room{}, err
	}
	if err := a.syncRoomGameState(); err != nil {
		return Room{}, err
	}

	return a.room.Snapshot(), nil
}

func (a *RoomActor) handleTimeout() (Room, TimeoutAction, error) {
	if a.room.CurrentGame == nil || a.room.DeadlineAt.IsZero() {
		return a.room.Snapshot(), TimeoutActionNone, nil
	}

	now := time.Now().UTC()
	if now.Before(a.room.DeadlineAt) {
		return a.room.Snapshot(), TimeoutActionNone, nil
	}

	currentGame := a.room.CurrentGame
	seatIndex := currentGame.CurrentSeatIndex
	if seatIndex < 0 || seatIndex >= len(currentGame.Players) {
		return Room{}, TimeoutActionNone, ErrInvalidRoomConfig
	}

	switch currentGame.Phase {
	case game.GamePhaseBidding:
		if err := currentGame.PlaceBid(seatIndex, 0, a.rng); err != nil {
			return Room{}, TimeoutActionNone, err
		}
		if err := a.syncRoomGameState(); err != nil {
			return Room{}, TimeoutActionNone, err
		}
		return a.room.Snapshot(), TimeoutActionAutoBid, nil
	case game.GamePhasePlaying:
		if currentGame.LastPlay != nil {
			if err := currentGame.Pass(seatIndex); err != nil {
				return Room{}, TimeoutActionNone, err
			}
			if err := a.syncRoomGameState(); err != nil {
				return Room{}, TimeoutActionNone, err
			}
			return a.room.Snapshot(), TimeoutActionAutoPass, nil
		}

		player := currentGame.Players[seatIndex]
		moves := game.LegalMoves(player.Hand, nil)
		if len(moves) == 0 {
			return Room{}, TimeoutActionNone, game.ErrInvalidCardSet
		}
		if err := currentGame.PlayCards(seatIndex, moves[0].Cards); err != nil {
			return Room{}, TimeoutActionNone, err
		}
		if err := a.syncRoomGameState(); err != nil {
			return Room{}, TimeoutActionNone, err
		}
		return a.room.Snapshot(), TimeoutActionAutoPlay, nil
	default:
		return a.room.Snapshot(), TimeoutActionNone, nil
	}
}

func (a *RoomActor) tryStartGame() (bool, error) {
	if a.room.Status != RoomStatusWaiting {
		return false, nil
	}
	if len(a.room.Seats) != a.room.MaxPlayers {
		return false, nil
	}
	for _, seat := range a.room.Seats {
		if !seat.Ready {
			return false, nil
		}
	}

	players := make([]game.GamePlayerInput, 0, len(a.room.Seats))
	for _, seat := range a.room.Seats {
		players = append(players, game.GamePlayerInput{
			UserID:    seat.UserID,
			SeatIndex: seat.SeatIndex,
			IsRobot:   seat.IsRobot,
		})
	}

	gameID := "g_" + a.room.ID
	currentGame, err := game.NewGame(gameID, players, a.rng)
	if err != nil {
		return false, err
	}

	a.room.Status = RoomStatusPlaying
	a.room.CurrentGame = currentGame
	if err := a.syncRoomGameState(); err != nil {
		return false, err
	}
	return true, nil
}

// syncRoomGameState 将游戏阶段同步回房间状态，并刷新下一回合截止时间。
func (a *RoomActor) syncRoomGameState() error {
	now := time.Now().UTC()
	a.room.UpdatedAt = now

	if a.room.CurrentGame == nil {
		a.room.DeadlineAt = time.Time{}
		return nil
	}

	switch a.room.CurrentGame.Phase {
	case game.GamePhaseBidding, game.GamePhasePlaying:
		a.room.Status = RoomStatusPlaying
		a.room.DeadlineAt = now.Add(defaultTurnTimeout)
	case game.GamePhaseEnded:
		a.room.Status = RoomStatusSettling
		a.room.DeadlineAt = time.Time{}
		_, err := a.room.CurrentGame.Settle(a.room.BaseScore)
		if err != nil {
			return err
		}
	default:
		a.room.DeadlineAt = time.Time{}
	}

	return nil
}
