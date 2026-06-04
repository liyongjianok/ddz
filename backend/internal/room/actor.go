package room

import (
	"errors"
	"time"

	"ddz/backend/internal/game"
)

var ErrRoomClosed = errors.New("room closed")

type roomCommandType string

const (
	roomCommandJoin     roomCommandType = "join"
	roomCommandLeave    roomCommandType = "leave"
	roomCommandReady    roomCommandType = "ready"
	roomCommandSnapshot roomCommandType = "snapshot"
)

type roomCommand struct {
	typ           roomCommandType
	userID        string
	preferredSeat *int
	ready         bool
	response      chan roomCommandResult
}

type roomCommandResult struct {
	room      Room
	seatIndex int
	started   bool
	err       error
}

// RoomActor 负责串行处理单个房间内的命令。
type RoomActor struct {
	room *Room
	rng  game.RNG
	cmds chan roomCommand
}

// NewRoomActor 创建单房间 Actor，并立即启动命令处理循环。
func NewRoomActor(room *Room, rng game.RNG) *RoomActor {
	actor := &RoomActor{
		room: room,
		rng:  rng,
		cmds: make(chan roomCommand),
	}

	go actor.loop()
	return actor
}

// Join 在房间 Actor 内串行处理加入请求。
func (a *RoomActor) Join(userID string, preferredSeat *int) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:           roomCommandJoin,
		userID:        userID,
		preferredSeat: preferredSeat,
	})
}

// Leave 在房间 Actor 内串行处理离开请求。
func (a *RoomActor) Leave(userID string) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:    roomCommandLeave,
		userID: userID,
	})
}

// Ready 在房间 Actor 内串行处理准备状态变更。
func (a *RoomActor) Ready(userID string, ready bool) (Room, int, bool, error) {
	return a.call(roomCommand{
		typ:    roomCommandReady,
		userID: userID,
		ready:  ready,
	})
}

// Snapshot 返回当前房间快照。
func (a *RoomActor) Snapshot() (Room, error) {
	result := make(chan roomCommandResult, 1)
	a.cmds <- roomCommand{
		typ:      roomCommandSnapshot,
		response: result,
	}
	reply := <-result
	return reply.room, reply.err
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
		case roomCommandSnapshot:
			result.room = a.room.Snapshot()
		default:
			result.err = ErrInvalidRoomConfig
		}
		cmd.response <- result
	}
}

func (a *RoomActor) handleJoin(userID string, preferredSeat *int) (Room, int, bool, error) {
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
	for i, seat := range a.room.Seats {
		if seat.UserID == userID {
			seatIndex = i
			break
		}
	}
	if seatIndex < 0 {
		return Room{}, -1, false, ErrUserNotInRoom
	}

	a.room.Seats = append(a.room.Seats[:seatIndex], a.room.Seats[seatIndex+1:]...)
	a.room.UpdatedAt = time.Now().UTC()
	if len(a.room.Seats) == 0 {
		a.room.Status = RoomStatusClosed
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
	a.room.UpdatedAt = time.Now().UTC()
	return true, nil
}
