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
