import type { RoomSnapshotSettlement } from "../../types/api";

interface SettlementPanelProps {
  settlement: RoomSnapshotSettlement;
  playerNames: Record<string, string>;
}

export function SettlementPanel({ settlement, playerNames }: SettlementPanelProps) {
  return (
    <section className="workspace-panel workspace-panel--full settlement-panel">
      <div className="panel-heading">
        <h2>结算</h2>
        <span className="status-pill">
          {settlement.winner_side === "landlord" ? "地主胜" : "农民胜"} · 倍数 {settlement.final_multiplier}
        </span>
      </div>
      <div className="settlement-grid">
        {settlement.players.map((player) => (
          <article key={player.user_id} className={`settlement-item${player.is_winner ? " is-winner" : ""}`}>
            <h3>{playerNames[player.user_id] ?? player.user_id}</h3>
            <p>{player.role === "landlord" ? "地主" : "农民"}</p>
            <strong>{player.score_delta > 0 ? `+${player.score_delta}` : player.score_delta}</strong>
          </article>
        ))}
      </div>
    </section>
  );
}
