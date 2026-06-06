import { env } from "../config/env";

export function buildRoomWSURL(roomID: string, token: string) {
  const baseURL = env.wsBaseURL.replace(/\/+$/, "");
  const query = new URLSearchParams({ token });
  return `${baseURL}/rooms/${roomID}?${query.toString()}`;
}
