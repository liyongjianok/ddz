import { requestJSON } from "./http";
import type { CurrentUserResponse, GuestLoginResponse } from "../types/api";

export function guestLogin(displayName: string) {
  return requestJSON<GuestLoginResponse>("/auth/guest", {
    method: "POST",
    body: JSON.stringify({
      display_name: displayName.trim(),
      avatar_url: "",
    }),
  });
}

export function fetchCurrentUser(token: string) {
  return requestJSON<CurrentUserResponse>("/auth/me", {
    method: "GET",
    token,
  });
}
