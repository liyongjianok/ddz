import { AuthProvider } from "../auth/AuthContext";
import { LobbyProvider } from "../lobby/LobbyContext";
import { AppRouter } from "../routes/AppRouter";

export function App() {
  return (
    <AuthProvider>
      <LobbyProvider>
        <AppRouter />
      </LobbyProvider>
    </AuthProvider>
  );
}
