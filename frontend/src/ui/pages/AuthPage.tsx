import { useMemo, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { AppLayout } from "../layout/AppLayout";

const guestNameSuggestions = ["牌桌老友", "春风得意", "稳稳出牌"];

export function AuthPage() {
  const auth = useAuth();
  const navigate = useNavigate();
  const [displayName, setDisplayName] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const helperText = useMemo(() => {
    if (auth.errorMessage) {
      return auth.errorMessage;
    }
    return localError;
  }, [auth.errorMessage, localError]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedName = displayName.trim();

    if (trimmedName.length === 0) {
      setLocalError("不填写昵称时，系统会自动生成游客名称");
    }

    if (trimmedName.length > 20) {
      setLocalError("昵称请控制在 20 个字符以内");
      return;
    }

    if (trimmedName.length > 0) {
      setLocalError(null);
    }
    try {
      await auth.loginAsGuest(trimmedName);
      navigate("/lobby", { replace: true });
    } catch {
      // login error already reflected in store
    }
  }

  return (
    <AppLayout>
      <section className="auth-hero">
        <div className="auth-hero__intro">
          <p className="auth-hero__eyebrow">欢乐斗地主</p>
          <h1 className="auth-hero__title">进入牌桌</h1>
          <p className="auth-hero__summary">
            先以游客身份进入大厅。当前任务把前端路由、认证状态和 API 调用链打通，后续大厅和房间再逐步补齐。
          </p>
        </div>

        <form className="auth-panel" onSubmit={handleSubmit}>
          <label className="field">
            <span className="field__label">游客昵称</span>
            <input
              autoComplete="nickname"
              className="field__input"
              maxLength={20}
              placeholder={guestNameSuggestions[0]}
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
            />
          </label>

          <div className="auth-suggestions">
            {guestNameSuggestions.map((name) => (
              <button
                key={name}
                className="chip-button"
                type="button"
                onClick={() => setDisplayName(name)}
              >
                {name}
              </button>
            ))}
          </div>

          <button className="primary-button" type="submit" disabled={auth.isLoggingIn}>
            {auth.isLoggingIn ? "登录中..." : "游客登录"}
          </button>

          <p className={`auth-panel__feedback${helperText ? " is-error" : ""}`}>
            {helperText ?? "不注册，直接进入大厅。"}
          </p>
        </form>
      </section>
    </AppLayout>
  );
}
