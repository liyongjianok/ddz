import { CardTile } from "./CardTile";
import type { RoomSnapshotPlayer, RoomSnapshotPlay } from "../../types/api";

interface PlayerSeatProps {
  player: RoomSnapshotPlayer;
  displayName: string;
  isCurrentTurn: boolean;
  isMe?: boolean;
  countdownText?: string;
  lastPlay?: RoomSnapshotPlay | null;
}

export function PlayerSeat({
  player,
  displayName,
  isCurrentTurn,
  isMe = false,
  countdownText,
  lastPlay,
}: PlayerSeatProps) {
  return (
    <section className={`player-seat${isCurrentTurn ? " is-current" : ""}${isMe ? " is-me" : ""}`}>
      <header className="player-seat__header">
        <div>
          <p className="player-seat__name">{displayName}</p>
          <p className="player-seat__meta">
            {renderRole(player.role)} · {renderPlayerStatus(player.status, player.ready)}
          </p>
        </div>
        <div className="player-seat__badges">
          {player.is_robot ? <span className="mini-badge">AI</span> : null}
          <span className={`mini-badge${player.status === "offline" ? " is-offline" : ""}`}>
            {player.status === "offline" ? "离线" : "在线"}
          </span>
        </div>
      </header>

      <div className="player-seat__counts">
        <span>座位 {player.seat_index + 1}</span>
        <strong>{player.remaining_count} 张</strong>
      </div>

      {countdownText ? <p className="player-seat__countdown">{countdownText}</p> : null}

      {lastPlay?.cards?.length ? (
        <div className="player-seat__last-play">
          {lastPlay.cards.map((card, index) => (
            <CardTile key={`${card}-${index}`} code={card} compact />
          ))}
        </div>
      ) : (
        <div className="player-seat__placeholder">暂未出牌</div>
      )}
    </section>
  );
}

function renderRole(role: string) {
  switch (role) {
    case "landlord":
      return "地主";
    case "farmer":
      return "农民";
    default:
      return "未定";
  }
}

function renderPlayerStatus(status: string, ready: boolean) {
  if (status === "offline") {
    return "离线";
  }
  if (status === "playing") {
    return "游戏中";
  }
  if (ready) {
    return "已准备";
  }
  return "待准备";
}
