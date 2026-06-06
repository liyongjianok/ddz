export interface APIResponse<T> {
  code: string;
  message: string;
  data: T;
  request_id?: string;
}

export interface APIErrorPayload {
  code: string;
  message: string;
  requestId?: string;
  status: number;
}

export interface GuestUser {
  id: string;
  display_name: string;
  avatar_url: string;
  account_type: string;
}

export interface GuestProfile {
  level: number;
  coin_balance: number;
  total_games: number;
  wins: number;
}

export interface GuestLoginResponse {
  user: GuestUser;
  access_token: string;
  expires_in: number;
}

export interface CurrentUserResponse {
  id: string;
  display_name: string;
  avatar_url: string;
  account_type: string;
  profile: GuestProfile;
}

export interface LobbyModeSummary {
  mode: string;
  base_score: number;
  online_players: number;
  waiting_rooms: number;
}

export interface LobbySummaryResponse {
  online_players: number;
  active_rooms: number;
  modes: LobbyModeSummary[];
}

export interface RoomListItem {
  room_id: string;
  mode: string;
  status: string;
  base_score: number;
  player_count: number;
  max_players: number;
  created_at: string;
}

export interface RoomListResponse {
  items: RoomListItem[];
  page: number;
  page_size: number;
  total: number;
}

export interface RoomAccessResponse {
  room_id: string;
  seat_index: number;
  ws_url: string;
}

export interface RoomListQuery {
  mode?: string;
  status?: string;
  page?: number;
  page_size?: number;
}
