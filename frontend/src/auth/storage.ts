const authStorageKey = "ddz_auth";

export interface StoredAuthState {
  token: string;
}

export function loadStoredAuthState(): StoredAuthState | null {
  try {
    const raw = window.localStorage.getItem(authStorageKey);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as StoredAuthState;
    if (!parsed.token) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function persistAuthState(state: StoredAuthState | null) {
  if (!state) {
    window.localStorage.removeItem(authStorageKey);
    return;
  }

  window.localStorage.setItem(authStorageKey, JSON.stringify(state));
}
