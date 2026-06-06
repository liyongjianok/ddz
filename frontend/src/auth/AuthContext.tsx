import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import { fetchCurrentUser, guestLogin } from "../api/auth";
import { APIError } from "../api/http";
import { loadStoredAuthState, persistAuthState } from "./storage";
import type { CurrentUserResponse } from "../types/api";

interface AuthContextValue {
  isReady: boolean;
  isAuthenticated: boolean;
  isLoggingIn: boolean;
  currentUser: CurrentUserResponse | null;
  accessToken: string | null;
  errorMessage: string | null;
  loginAsGuest: (displayName: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [isReady, setIsReady] = useState(false);
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const [currentUser, setCurrentUser] = useState<CurrentUserResponse | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function restoreSession() {
      const stored = loadStoredAuthState();
      if (!stored?.token) {
        setIsReady(true);
        return;
      }

      try {
        const user = await fetchCurrentUser(stored.token);
        if (cancelled) {
          return;
        }
        setAccessToken(stored.token);
        setCurrentUser(user);
      } catch {
        if (!cancelled) {
          persistAuthState(null);
        }
      } finally {
        if (!cancelled) {
          setIsReady(true);
        }
      }
    }

    void restoreSession();
    return () => {
      cancelled = true;
    };
  }, []);

  const loginAsGuest = useCallback(async (displayName: string) => {
    setIsLoggingIn(true);
    setErrorMessage(null);

    try {
      const result = await guestLogin(displayName);
      const user = await fetchCurrentUser(result.access_token);
      setAccessToken(result.access_token);
      setCurrentUser(user);
      persistAuthState({ token: result.access_token });
    } catch (error) {
      if (error instanceof APIError) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage("登录失败，请稍后重试");
      }
      throw error;
    } finally {
      setIsLoggingIn(false);
    }
  }, []);

  const logout = useCallback(() => {
    setAccessToken(null);
    setCurrentUser(null);
    setErrorMessage(null);
    persistAuthState(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      isReady,
      isAuthenticated: Boolean(accessToken && currentUser),
      isLoggingIn,
      currentUser,
      accessToken,
      errorMessage,
      loginAsGuest,
      logout,
    }),
    [accessToken, currentUser, errorMessage, isLoggingIn, isReady, loginAsGuest, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
