import { AuthProvider } from "../auth/AuthContext";
import { LobbyProvider } from "../lobby/LobbyContext";
import { RoomProvider } from "../room/RoomContext";
import { AppRouter } from "../routes/AppRouter";

export function App() {
  return (
    <AuthProvider>
      <LobbyProvider>
        <RoomProvider>
          <AppRouter />
        </RoomProvider>
      </LobbyProvider>
    </AuthProvider>
  );
}
