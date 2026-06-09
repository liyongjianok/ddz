import type {
  GameBidPlacedEvent,
  GameCardsPlayedEvent,
  GameEndedEvent,
  GameLandlordDecidedEvent,
  GameMyHandUpdatedEvent,
  GamePlayerPassedEvent,
  RoomPlayerReadyEvent,
  RoomSnapshot,
  WSAckPayload,
  WSEnvelope,
  WSErrorPayload,
} from "../types/api";

export type RoomConnectionState =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected"
  | "failed";

export type RoomClientMessageType = "ping" | "room.ready" | "game.bid" | "game.play_cards" | "game.pass";

export type RoomServerMessage =
  | WSEnvelope<RoomSnapshot>
  | WSEnvelope<WSAckPayload>
  | WSEnvelope<WSErrorPayload>
  | WSEnvelope<RoomPlayerReadyEvent>
  | WSEnvelope<GameBidPlacedEvent>
  | WSEnvelope<GameLandlordDecidedEvent>
  | WSEnvelope<GameCardsPlayedEvent>
  | WSEnvelope<GameMyHandUpdatedEvent>
  | WSEnvelope<GamePlayerPassedEvent>
  | WSEnvelope<GameEndedEvent>;

export interface RoomClientMessage<TPayload> {
  type: RoomClientMessageType;
  request_id: string;
  seq: number;
  payload: TPayload;
}
