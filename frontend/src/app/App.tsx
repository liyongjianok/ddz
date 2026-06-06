import { AuthProvider } from "../auth/AuthContext";
import { AppRouter } from "../routes/AppRouter";

export function App() {
  return (
    <AuthProvider>
      <AppRouter />
    </AuthProvider>
  );
}
