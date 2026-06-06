import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { AppLayout } from "../ui/layout/AppLayout";
import { AuthPage } from "../ui/pages/AuthPage";
import { LobbyPage } from "../ui/pages/LobbyPage";
import { RoomPage } from "../ui/pages/RoomPage";

function ProtectedRoute({ children }: { children: JSX.Element }) {
  const auth = useAuth();

  if (!auth.isReady) {
    return <AppLayout tone="loading">正在恢复登录状态...</AppLayout>;
  }
  if (!auth.isAuthenticated) {
    return <Navigate to="/auth" replace />;
  }
  return children;
}

function AuthRoute() {
  const auth = useAuth();

  if (!auth.isReady) {
    return <AppLayout tone="loading">正在初始化前端...</AppLayout>;
  }
  if (auth.isAuthenticated) {
    return <Navigate to="/lobby" replace />;
  }
  return <AuthPage />;
}

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/auth" replace />} />
        <Route path="/auth" element={<AuthRoute />} />
        <Route
          path="/lobby"
          element={
            <ProtectedRoute>
              <LobbyPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/rooms/:roomID"
          element={
            <ProtectedRoute>
              <RoomPage />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/auth" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
