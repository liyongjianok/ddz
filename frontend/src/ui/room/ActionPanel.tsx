interface ActionPanelProps {
  phase: string;
  isMyTurn: boolean;
  canPass: boolean;
  selectedCount: number;
  canPlaySelected: boolean;
  selectionHint: string;
  canHint: boolean;
  isWaitingRoom: boolean;
  isReady: boolean;
  onReady: () => void;
  onBid: (score: 0 | 1 | 2 | 3) => void;
  onPlay: () => void;
  onPass: () => void;
  onHint: () => void;
  onClear: () => void;
}

export function ActionPanel({
  phase,
  isMyTurn,
  canPass,
  selectedCount,
  canPlaySelected,
  selectionHint,
  canHint,
  isWaitingRoom,
  isReady,
  onReady,
  onBid,
  onPlay,
  onPass,
  onHint,
  onClear,
}: ActionPanelProps) {
  if (isWaitingRoom) {
    return (
      <section className="action-panel">
        <div className="action-panel__header">
          <h2>房间准备</h2>
          <span className="panel-hint">{isReady ? "你已准备，等待其他玩家。" : "满 3 人且全部准备后自动开局。"}</span>
        </div>
        <div className="action-row">
          <button className="primary-button" type="button" onClick={onReady} disabled={isReady}>
            {isReady ? "已准备" : "准备开始"}
          </button>
        </div>
      </section>
    );
  }

  if (phase === "bidding") {
    return (
      <section className="action-panel">
        <div className="action-panel__header">
          <h2>叫分阶段</h2>
          <span className="panel-hint">{isMyTurn ? "轮到你叫分" : "等待当前玩家叫分"}</span>
        </div>
        <div className="action-row action-row--grid">
          {[0, 1, 2, 3].map((score) => (
            <button
              key={score}
              className="secondary-button"
              type="button"
              disabled={!isMyTurn}
              onClick={() => onBid(score as 0 | 1 | 2 | 3)}
            >
              {score === 0 ? "不叫" : `叫 ${score} 分`}
            </button>
          ))}
        </div>
      </section>
    );
  }

  if (phase === "playing") {
    return (
      <section className="action-panel">
        <div className="action-panel__header">
          <h2>出牌阶段</h2>
          <span className={`panel-hint${isMyTurn && selectedCount > 0 && !canPlaySelected ? " is-warning" : ""}`}>
            {isMyTurn ? selectionHint : "等待当前玩家操作"}
          </span>
        </div>
        <div className="action-row">
          <button className="secondary-button" type="button" onClick={onClear} disabled={selectedCount === 0}>
            清空选择
          </button>
          <button className="secondary-button" type="button" onClick={onHint} disabled={!isMyTurn || !canHint}>
            提示
          </button>
          <button className="secondary-button" type="button" onClick={onPass} disabled={!isMyTurn || !canPass}>
            不出
          </button>
          <button className="primary-button" type="button" onClick={onPlay} disabled={!isMyTurn || !canPlaySelected}>
            出选中牌
          </button>
        </div>
      </section>
    );
  }

  return (
    <section className="action-panel">
      <div className="action-panel__header">
        <h2>本局结束</h2>
        <span className="panel-hint">等待下一次准备</span>
      </div>
    </section>
  );
}
