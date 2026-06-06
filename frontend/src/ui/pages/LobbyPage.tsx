import { Link } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { AppLayout } from "../layout/AppLayout";

export function LobbyPage() {
  const auth = useAuth();

  return (
    <AppLayout>
      <section className="workspace-page">
        <header className="workspace-header">
          <div>
            <p className="workspace-header__eyebrow">大厅骨架</p>
            <h1 className="workspace-header__title">欢迎，{auth.currentUser?.display_name ?? "游客"}</h1>
          </div>
          <button className="secondary-button" type="button" onClick={auth.logout}>
            退出登录
          </button>
        </header>

        <div className="workspace-grid">
          <section className="workspace-panel">
            <h2>当前进度</h2>
            <p>API client、路由和认证状态已接通。下一步会补大厅摘要、快速开始和房间列表。</p>
          </section>

          <section className="workspace-panel">
            <h2>登录态验证</h2>
            <dl className="meta-list">
              <div>
                <dt>用户 ID</dt>
                <dd>{auth.currentUser?.id ?? "-"}</dd>
              </div>
              <div>
                <dt>账户类型</dt>
                <dd>{auth.currentUser?.account_type ?? "-"}</dd>
              </div>
              <div>
                <dt>金币</dt>
                <dd>{auth.currentUser?.profile.coin_balance ?? 0}</dd>
              </div>
            </dl>
          </section>

          <section className="workspace-panel workspace-panel--full">
            <h2>下一站</h2>
            <div className="route-links">
              <Link className="link-card" to="/rooms/demo-room">
                房间路由占位
              </Link>
            </div>
          </section>
        </div>
      </section>
    </AppLayout>
  );
}
