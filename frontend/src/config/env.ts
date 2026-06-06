const defaultAPIBaseURL = "/api/v1";

function resolveDefaultWSBaseURL() {
  if (typeof window === "undefined") {
    return "ws://127.0.0.1:8080/ws/v1";
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws/v1`;
}

export const env = {
  apiBaseURL: import.meta.env.VITE_API_BASE_URL ?? defaultAPIBaseURL,
  wsBaseURL: import.meta.env.VITE_WS_BASE_URL ?? resolveDefaultWSBaseURL(),
};
