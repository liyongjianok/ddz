import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { useLobby } from "../../lobby/LobbyContext";
import { AppLayout } from "../layout/AppLayout";

export function LobbyPage() {
  const auth = useAuth();
  const lobby = useLobby();
  const navigate = useNavigate();

  const modeOptions = useMemo(() => {
    if (lobby.summary?.modes.length) {
      return lobby.summary.modes;
    }
    return [
      {
        mode: "classic",
        base_score: 1,
        online_players: 0,
        waiting_rooms: 0,
      },
    ];
  }, [lobby.summary?.modes]);

  async function handleQuickStart() {
    try {
      const access = await lobby.quickStartGame();
      navigate(`/rooms/${access.room_id}`);
    } catch {
      // error is reflected by store
    }
  }

  async function handleJoinRoom(roomID: string) {
    try {
      const access = await lobby.joinExistingRoom(roomID);
      navigate(`/rooms/${access.room_id}`);
    } catch {
      // error is reflected by store
    }
  }

  return (
    <AppLayout>
      <section className="workspace-page">
        <header className="workspace-header">
          <div>
            <p className="workspace-header__eyebrow">经典场大厅</p>
            <h1 className="workspace-header__title">欢迎，{auth.currentUser?.display_name ?? "游客"}</h1>
          </div>
          <div className="workspace-header__actions">
            <button className="secondary-button" type="button" onClick={() => void lobby.refreshLobby()}>
              {lobby.isRefreshing ? "刷新中..." : "刷新大厅"}
            </button>
            <button className="secondary-button" type="button" onClick={auth.logout}>
              退出登录
            </button>
          </div>
        </header>

        <div className="workspace-grid">
          <section className="workspace-panel lobby-summary">
            <h2>大厅摘要</h2>
            <div className="summary-metrics">
              <article className="summary-metric">
                <span className="summary-metric__label">在线人数</span>
                <strong className="summary-metric__value">{lobby.summary?.online_players ?? 0}</strong>
              </article>
              <article className="summary-metric">
                <span className="summary-metric__label">活跃房间</span>
                <strong className="summary-metric__value">{lobby.summary?.active_rooms ?? 0}</strong>
              </article>
              <article className="summary-metric">
                <span className="summary-metric__label">当前金币</span>
                <strong className="summary-metric__value">
                  {auth.currentUser?.profile.coin_balance ?? 0}
                </strong>
              </article>
            </div>
          </section>

          <section className="workspace-panel quick-start-panel">
            <div className="panel-heading">
              <h2>快速开始</h2>
              <span className="status-pill">经典模式</span>
            </div>

            <div className="score-selector" role="radiogroup" aria-label="选择底分">
              {modeOptions.map((option) => (
                <button
                  key={`${option.mode}-${option.base_score}`}
                  className={`score-option${
                    lobby.selectedBaseScore === option.base_score ? " is-active" : ""
                  }`}
                  type="button"
                  onClick={() => lobby.setSelectedBaseScore(option.base_score)}
                >
                  <span className="score-option__title">{option.base_score} 分场</span>
                  <span className="score-option__meta">
                    {option.online_players} 人在线 · {option.waiting_rooms} 个等待房间
                  </span>
                </button>
              ))}
            </div>

            <button
              className="primary-button quick-start-button"
              type="button"
              disabled={lobby.isQuickStarting}
              onClick={() => void handleQuickStart()}
            >
              {lobby.isQuickStarting ? "正在分配房间..." : "快速开始"}
            </button>
            <p className="panel-hint">优先进入等待中的可用房间，否则创建新房间。</p>
          </section>

          <section className="workspace-panel workspace-panel--full">
            <div className="panel-heading">
              <h2>房间列表</h2>
              <div className="filter-group" role="tablist" aria-label="房间状态过滤">
                {[
                  { label: "全部", value: "all" },
                  { label: "等待中", value: "waiting" },
                  { label: "对局中", value: "playing" },
                ].map((filter) => (
                  <button
                    key={filter.value}
                    className={`filter-chip${
                      lobby.selectedStatus === filter.value ? " is-active" : ""
                    }`}
                    type="button"
                    onClick={() => lobby.setSelectedStatus(filter.value as "all" | "waiting" | "playing")}
                  >
                    {filter.label}
                  </button>
                ))}
              </div>
            </div>

            {lobby.errorMessage ? <p className="panel-error">{lobby.errorMessage}</p> : null}

            <div className="room-list-meta">
              <span>共 {lobby.totalRooms} 个公开房间</span>
              <span>当前账号：{auth.currentUser?.account_type ?? "-"}</span>
            </div>

            <div className="room-list">
              {lobby.isLoading ? (
                <div className="empty-state">正在加载大厅数据...</div>
              ) : lobby.rooms.length === 0 ? (
                <div className="empty-state">当前筛选下暂无房间，直接快速开始即可开局。</div>
              ) : (
                lobby.rooms.map((room) => (
                  <article key={room.room_id} className="room-item">
                    <div className="room-item__main">
                      <div>
                        <p className="room-item__eyebrow">房间号 {room.room_id}</p>
                        <h3 className="room-item__title">
                          {room.base_score} 分场 · {room.mode === "classic" ? "经典模式" : room.mode}
                        </h3>
                      </div>
                      <span className={`status-pill status-pill--${room.status}`}>{renderRoomStatus(room.status)}</span>
                    </div>

                    <dl className="room-item__meta">
                      <div>
                        <dt>人数</dt>
                        <dd>
                          {room.player_count}/{room.max_players}
                        </dd>
                      </div>
                      <div>
                        <dt>创建时间</dt>
                        <dd>{formatCreatedAt(room.created_at)}</dd>
                      </div>
                    </dl>

                    <button
                      className="secondary-button room-item__action"
                      type="button"
                      disabled={room.status !== "waiting" || lobby.joiningRoomID === room.room_id}
                      onClick={() => void handleJoinRoom(room.room_id)}
                    >
                      {lobby.joiningRoomID === room.room_id ? "进入中..." : "进入房间"}
                    </button>
                  </article>
                ))
              )}
            </div>
          </section>
        </div>
      </section>
    </AppLayout>
  );
}

function renderRoomStatus(status: string) {
  switch (status) {
    case "waiting":
      return "等待中";
    case "playing":
      return "对局中";
    case "settling":
      return "结算中";
    default:
      return status;
  }
}

function formatCreatedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}
