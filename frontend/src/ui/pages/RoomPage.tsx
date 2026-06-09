import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { useRoom } from "../../room/RoomContext";
import type { RoomSnapshotPlay, RoomSnapshotPlayer } from "../../types/api";
import { AppLayout } from "../layout/AppLayout";
import { ActionPanel } from "../room/ActionPanel";
import { CardTile } from "../room/CardTile";
import { HandCards } from "../room/HandCards";
import { PlayerSeat } from "../room/PlayerSeat";
import { SettlementPanel } from "../room/SettlementPanel";

export function RoomPage() {
  const { roomID = "" } = useParams();
  const auth = useAuth();
  const room = useRoom();

  useEffect(() => {
    if (!roomID || !auth.accessToken) {
      return;
    }
    room.connectRoom(roomID);
    return () => {
      room.disconnectRoom();
    };
  }, [auth.accessToken, room.connectRoom, room.disconnectRoom, roomID]);

  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);
    return () => {
      window.clearInterval(timer);
    };
  }, []);

  const playerNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const player of room.players) {
      map[player.user_id] = player.user_id === auth.currentUser?.id ? auth.currentUser.display_name : `玩家${player.seat_index + 1}`;
    }
    return map;
  }, [auth.currentUser?.display_name, auth.currentUser?.id, room.players]);

  const mySeat = room.me?.seat_index ?? -1;
  const isWaitingRoom = !room.game;
  const currentSeatIndex = room.game?.current_seat_index ?? -1;
  const isMyTurn = mySeat >= 0 && currentSeatIndex === mySeat;
  const lastPlaySeatIndex = room.game?.last_play?.seat_index ?? -1;
  const canPass = room.game?.phase === "playing" && Boolean(room.game.last_play) && lastPlaySeatIndex !== mySeat;
  const myPlayer = room.players.find((player) => player.seat_index === mySeat);
  const selectedHand = room.me?.hand ?? [];
  const deadlineText = formatDeadline(room.game?.deadline_at, now);

  return (
    <AppLayout>
      <section className="workspace-page room-page">
        <header className="workspace-header">
          <div>
            <p className="workspace-header__eyebrow">
              房间 {(room.snapshot?.room.room_id ?? roomID) || "-"} · {renderPhase(room.game?.phase, Boolean(room.game))}
            </p>
            <h1 className="workspace-header__title">房间 {roomID || "-"}</h1>
          </div>
          <div className="workspace-header__actions">
            <span className={`status-pill status-pill--${room.connectionState}`}>{renderConnectionState(room.connectionState)}</span>
            <Link className="secondary-link" to="/lobby">
              返回大厅
            </Link>
          </div>
        </header>

        <div className="room-board">
          <section className="workspace-panel workspace-panel--full room-overview">
            <div className="room-overview__meta">
              <span>底分 {room.snapshot?.room.base_score ?? "-"}</span>
              <span>倍数 {room.game?.multiplier ?? 1}</span>
              <span>{deadlineText}</span>
            </div>
            {room.errorMessage ? <p className="panel-error">{room.errorMessage}</p> : null}
          </section>

          <div className="room-table">
            <div className="room-table__seat room-table__seat--top">
              {renderSeat(room.players, 2, playerNameMap, room.game?.last_play, currentSeatIndex, deadlineText, mySeat)}
            </div>

            <div className="room-table__seat room-table__seat--left">
              {renderSeat(room.players, 1, playerNameMap, room.game?.last_play, currentSeatIndex, deadlineText, mySeat)}
            </div>

            <section className="workspace-panel room-table__center">
              <div className="panel-heading">
                <h2>牌桌中央</h2>
                <span className="panel-hint">
                  {typeof room.game?.landlord_seat_index === "number" && room.game.landlord_seat_index >= 0
                    ? `地主座位 ${room.game.landlord_seat_index + 1}`
                    : "等待地主确定"}
                </span>
              </div>

              <div className="bottom-cards">
                {(room.game?.bottom_cards?.length ? room.game.bottom_cards : ["back", "back", "back"]).map((card, index) => (
                  <CardTile key={`${card}-${index}`} code={card} faceDown={card === "back"} compact />
                ))}
              </div>

              <div className="last-play-zone">
                {room.game?.last_play?.cards?.length ? (
                  room.game.last_play.cards.map((card, index) => (
                    <CardTile key={`${card}-${index}`} code={card} compact />
                  ))
                ) : (
                  <div className="empty-state empty-state--compact">本轮尚未出牌</div>
                )}
              </div>

              <div className="room-table__tip">
                {isWaitingRoom
                  ? "等待房间满员并全部准备"
                  : isMyTurn
                    ? "轮到你操作"
                    : `当前轮到座位 ${currentSeatIndex + 1}`}
              </div>
            </section>

            <div className="room-table__seat room-table__seat--right">
              {renderSeat(room.players, 0, playerNameMap, room.game?.last_play, currentSeatIndex, deadlineText, mySeat)}
            </div>
          </div>

          <section className="workspace-panel workspace-panel--full my-hand-panel">
            <div className="panel-heading">
              <h2>我的手牌</h2>
              <span className="panel-hint">{selectedHand.length} 张</span>
            </div>
            <HandCards
              cards={selectedHand}
              selectedCards={room.selectedCards}
              disabled={!room.game || room.game.phase === "ended"}
              onToggle={room.toggleCard}
            />
          </section>

          <ActionPanel
            phase={room.game?.phase ?? "waiting"}
            isMyTurn={isMyTurn}
            canPass={canPass}
            selectedCount={room.selectedCards.length}
            isWaitingRoom={isWaitingRoom}
            isReady={Boolean(myPlayer?.ready)}
            onReady={() => room.sendReady(true)}
            onBid={room.sendBid}
            onPlay={room.sendPlay}
            onPass={room.sendPass}
            onClear={room.clearSelection}
          />

          {room.game?.settlement ? (
            <SettlementPanel settlement={room.game.settlement} playerNames={playerNameMap} />
          ) : null}
        </div>
      </section>
    </AppLayout>
  );
}

function renderSeat(
  players: RoomSnapshotPlayer[],
  seatIndex: number,
  playerNameMap: Record<string, string>,
  lastPlay: RoomSnapshotPlay | undefined,
  currentSeatIndex: number,
  deadlineText: string,
  mySeat: number,
) {
  const player = players.find((item) => item.seat_index === seatIndex);
  if (!player) {
    return <section className="player-seat player-seat--empty">等待玩家加入</section>;
  }

  return (
    <PlayerSeat
      player={player}
      displayName={playerNameMap[player.user_id] ?? `玩家${player.seat_index + 1}`}
      isCurrentTurn={player.seat_index === currentSeatIndex}
      isMe={player.seat_index === mySeat}
      countdownText={player.seat_index === currentSeatIndex ? deadlineText : undefined}
      lastPlay={lastPlay?.seat_index === player.seat_index ? lastPlay : null}
    />
  );
}

function renderPhase(phase?: string, started = false) {
  if (!started) {
    return "等待中";
  }
  switch (phase) {
    case "bidding":
      return "叫分中";
    case "playing":
      return "出牌中";
    case "ended":
      return "已结算";
    default:
      return phase ?? "房间中";
  }
}

function renderConnectionState(state: string) {
  switch (state) {
    case "connected":
      return "已连接";
    case "connecting":
      return "连接中";
    case "reconnecting":
      return "重连中";
    case "failed":
      return "连接失败";
    case "disconnected":
      return "已断开";
    default:
      return "未连接";
  }
}

function formatDeadline(value: string | undefined, now: number) {
  if (!value) {
    return "等待计时";
  }
  const deadline = new Date(value);
  if (Number.isNaN(deadline.getTime())) {
    return "等待计时";
  }

  const remainingSeconds = Math.max(0, Math.ceil((deadline.getTime() - now) / 1000));
  if (remainingSeconds > 0) {
    return `剩余 ${remainingSeconds} 秒`;
  }

  return "等待服务器推进";
}
