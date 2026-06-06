import { requestJSON } from "./http";
import type {
  LobbySummaryResponse,
  RoomAccessResponse,
  RoomListQuery,
  RoomListResponse,
} from "../types/api";

function buildRoomListQuery(query: RoomListQuery) {
  const params = new URLSearchParams();

  if (query.mode) {
    params.set("mode", query.mode);
  }
  if (query.status) {
    params.set("status", query.status);
  }
  if (query.page) {
    params.set("page", String(query.page));
  }
  if (query.page_size) {
    params.set("page_size", String(query.page_size));
  }

  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

export function fetchLobbySummary(token: string) {
  return requestJSON<LobbySummaryResponse>("/lobby/summary", {
    method: "GET",
    token,
  });
}

export function fetchRoomList(token: string, query: RoomListQuery) {
  return requestJSON<RoomListResponse>(`/rooms${buildRoomListQuery(query)}`, {
    method: "GET",
    token,
  });
}

export function quickStart(token: string, baseScore: number) {
  return requestJSON<RoomAccessResponse>("/matchmaking/quick-start", {
    method: "POST",
    token,
    body: JSON.stringify({
      mode: "classic",
      base_score: baseScore,
    }),
  });
}

export function joinRoom(token: string, roomID: string) {
  return requestJSON<RoomAccessResponse>(`/rooms/${roomID}/join`, {
    method: "POST",
    token,
    body: JSON.stringify({}),
  });
}
