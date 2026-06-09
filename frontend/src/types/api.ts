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

export interface RoomSnapshotRoom {
  room_id: string;
  mode: string;
  status: string;
  base_score: number;
}

export interface RoomSnapshotPlayer {
  user_id: string;
  seat_index: number;
  role: string;
  status: string;
  ready: boolean;
  remaining_count: number;
  is_robot: boolean;
}

export interface RoomSnapshotCardGroup {
  type: string;
  rank: string;
  length: number;
  attachments?: string[];
}

export interface RoomSnapshotPlay {
  seat_index: number;
  user_id: string;
  cards: string[];
  card_group: RoomSnapshotCardGroup;
  created_at: string;
}

export interface RoomSnapshotSettlementPlayer {
  user_id: string;
  seat_index: number;
  role: string;
  score_delta: number;
  is_winner: boolean;
}

export interface RoomSnapshotSettlement {
  winner_side: string;
  final_multiplier: number;
  base_score: number;
  players: RoomSnapshotSettlementPlayer[];
}

export interface RoomSnapshotGame {
  game_id: string;
  phase: string;
  current_seat_index: number;
  landlord_seat_index: number;
  bottom_cards?: string[];
  last_play?: RoomSnapshotPlay;
  multiplier: number;
  deadline_at?: string;
  settlement?: RoomSnapshotSettlement;
}

export interface RoomSnapshotMe {
  user_id: string;
  seat_index: number;
  hand: string[];
}

export interface RoomSnapshot {
  room: RoomSnapshotRoom;
  players: RoomSnapshotPlayer[];
  game?: RoomSnapshotGame;
  me: RoomSnapshotMe;
}

export interface WSEnvelope<TPayload> {
  type: string;
  request_id: string | null;
  seq: number;
  server_time: string;
  payload: TPayload;
}

export interface WSAckPayload {
  ok: boolean;
}

export interface WSErrorPayload {
  code: string;
  message: string;
}

export interface RoomPlayerReadyEvent {
  user_id: string;
  seat_index: number;
  ready: boolean;
}

export interface GameBidPlacedEvent {
  user_id: string;
  seat_index: number;
  score: number;
  next_seat_index: number;
  deadline_at?: string;
}

export interface GameLandlordDecidedEvent {
  landlord_seat_index: number;
  landlord_user_id: string;
  bottom_cards: string[];
  multiplier: number;
  current_seat_index: number;
  deadline_at?: string;
}

export interface GameCardsPlayedEvent {
  user_id: string;
  seat_index: number;
  cards: string[];
  card_group: RoomSnapshotCardGroup;
  remaining_count: number;
  next_seat_index: number;
  deadline_at?: string;
}

export interface GameMyHandUpdatedEvent {
  cards: string[];
}

export interface GamePlayerPassedEvent {
  user_id: string;
  seat_index: number;
  next_seat_index: number;
  deadline_at?: string;
}

export interface GameEndedSettlementEvent {
  user_id: string;
  seat_index: number;
  role: string;
  score_delta: number;
}

export interface GameEndedEvent {
  winner_side: string;
  winner_user_id: string;
  settlements: GameEndedSettlementEvent[];
  final_multiplier: number;
}
