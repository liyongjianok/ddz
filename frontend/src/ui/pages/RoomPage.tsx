import { useMemo } from "react";
import { Link, useParams } from "react-router-dom";
import { buildRoomWSURL } from "../../api/ws";
import { useAuth } from "../../auth/AuthContext";
import { AppLayout } from "../layout/AppLayout";

export function RoomPage() {
  const { roomID = "" } = useParams();
  const auth = useAuth();

  const wsURL = useMemo(() => {
    if (!auth.accessToken || !roomID) {
      return "";
    }
    return buildRoomWSURL(roomID, auth.accessToken);
  }, [auth.accessToken, roomID]);

  return (
    <AppLayout>
      <section className="workspace-page">
        <header className="workspace-header">
          <div>
            <p className="workspace-header__eyebrow">房间路由骨架</p>
            <h1 className="workspace-header__title">房间 {roomID || "-"}</h1>
          </div>
          <Link className="secondary-link" to="/lobby">
            返回大厅
          </Link>
        </header>

        <div className="workspace-grid">
          <section className="workspace-panel workspace-panel--full">
            <h2>WebSocket 地址预览</h2>
            <code className="inline-code">{wsURL || "未生成"}</code>
          </section>

          <section className="workspace-panel">
            <h2>说明</h2>
            <p>当前任务只做到前端骨架和游客登录接通，真实房间渲染将在后续任务补上。</p>
          </section>
        </div>
      </section>
    </AppLayout>
  );
}
