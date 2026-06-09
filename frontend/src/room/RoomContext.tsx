import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PropsWithChildren,
} from "react";
import { buildRoomWSURL } from "../api/ws";
import { useAuth } from "../auth/AuthContext";
import type {
  GameBidPlacedEvent,
  GameCardsPlayedEvent,
  GameEndedEvent,
  GameLandlordDecidedEvent,
  GameMyHandUpdatedEvent,
  GamePlayerPassedEvent,
  RoomPlayerReadyEvent,
  RoomSnapshot,
  RoomSnapshotGame,
  RoomSnapshotPlayer,
  WSErrorPayload,
} from "../types/api";
import { RoomSocketClient } from "./RoomSocketClient";
import type { RoomConnectionState, RoomServerMessage } from "./roomTypes";

interface RoomContextValue {
  snapshot: RoomSnapshot | null;
  players: RoomSnapshotPlayer[];
  me: RoomSnapshot["me"] | null;
  game: RoomSnapshotGame | null;
  connectionState: RoomConnectionState;
  errorMessage: string | null;
  selectedCards: string[];
  roomID: string | null;
  connectRoom: (roomID: string) => void;
  disconnectRoom: () => void;
  toggleCard: (card: string) => void;
  clearSelection: () => void;
  sendReady: (ready: boolean) => void;
  sendBid: (score: 0 | 1 | 2 | 3) => void;
  sendPlay: () => void;
  sendPass: () => void;
}

const RoomContext = createContext<RoomContextValue | null>(null);

function updatePlayer(
  players: RoomSnapshotPlayer[],
  seatIndex: number,
  updater: (player: RoomSnapshotPlayer) => RoomSnapshotPlayer,
) {
  return players.map((player) => {
    if (player.seat_index !== seatIndex) {
      return player;
    }
    return updater(player);
  });
}

function sortSelectedCards(current: string[], cards: string[]) {
  return cards.filter((card) => current.includes(card));
}

export function RoomProvider({ children }: PropsWithChildren) {
  const auth = useAuth();
  const clientRef = useRef<RoomSocketClient | null>(null);
  const connectionVersionRef = useRef(0);
  const [roomID, setRoomID] = useState<string | null>(null);
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [connectionState, setConnectionState] = useState<RoomConnectionState>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [selectedCards, setSelectedCards] = useState<string[]>([]);

  const sendMessage = useCallback(
    <TPayload,>(type: "room.ready" | "game.bid" | "game.play_cards" | "game.pass", payload: TPayload) => {
      try {
        setErrorMessage(null);
        clientRef.current?.send(type, payload);
      } catch {
        setErrorMessage("房间连接尚未就绪，请稍后重试");
      }
    },
    [],
  );

  const handleServerMessage = useCallback((message: RoomServerMessage) => {
    switch (message.type) {
      case "room.snapshot": {
        const nextSnapshot = message.payload as RoomSnapshot;
        setSnapshot(nextSnapshot);
        setSelectedCards((current) => sortSelectedCards(current, nextSnapshot.me.hand ?? []));
        return;
      }
      case "error": {
        const payload = message.payload as WSErrorPayload;
        setErrorMessage(payload.message);
        return;
      }
      case "room.player_ready": {
        const payload = message.payload as RoomPlayerReadyEvent;
        setSnapshot((current) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            players: updatePlayer(current.players, payload.seat_index, (player) => ({
              ...player,
              ready: payload.ready,
              status: payload.ready ? "ready" : "joined",
            })),
          };
        });
        return;
      }
      case "game.bid_placed": {
        const payload = message.payload as GameBidPlacedEvent;
        setSnapshot((current) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            game: current.game
              ? {
                  ...current.game,
                  current_seat_index: payload.next_seat_index,
                  deadline_at: payload.deadline_at,
                }
              : current.game,
          };
        });
        return;
      }
      case "game.landlord_decided": {
        const payload = message.payload as GameLandlordDecidedEvent;
        setSnapshot((current) => {
          if (!current || !current.game) {
            return current;
          }

          return {
            ...current,
            players: current.players.map((player) => ({
              ...player,
              role:
                player.seat_index === payload.landlord_seat_index
                  ? "landlord"
                  : player.role
                    ? player.role
                    : "farmer",
            })),
            game: {
              ...current.game,
              phase: "playing",
              landlord_seat_index: payload.landlord_seat_index,
              bottom_cards: payload.bottom_cards,
              multiplier: payload.multiplier,
              current_seat_index: payload.current_seat_index,
              deadline_at: payload.deadline_at,
            },
          };
        });
        return;
      }
      case "game.cards_played": {
        const payload = message.payload as GameCardsPlayedEvent;
        setSnapshot((current) => {
          if (!current || !current.game) {
            return current;
          }

          return {
            ...current,
            players: updatePlayer(current.players, payload.seat_index, (player) => ({
              ...player,
              remaining_count: payload.remaining_count,
            })),
            game: {
              ...current.game,
              current_seat_index: payload.next_seat_index,
              deadline_at: payload.deadline_at,
              last_play: {
                seat_index: payload.seat_index,
                user_id: payload.user_id,
                cards: payload.cards,
                card_group: payload.card_group,
                created_at: new Date().toISOString(),
              },
            },
          };
        });
        return;
      }
      case "game.my_hand_updated": {
        const payload = message.payload as GameMyHandUpdatedEvent;
        setSnapshot((current) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            me: {
              ...current.me,
              hand: payload.cards,
            },
          };
        });
        setSelectedCards([]);
        return;
      }
      case "game.player_passed": {
        const payload = message.payload as GamePlayerPassedEvent;
        setSnapshot((current) => {
          if (!current || !current.game) {
            return current;
          }
          return {
            ...current,
            game: {
              ...current.game,
              current_seat_index: payload.next_seat_index,
              deadline_at: payload.deadline_at,
            },
          };
        });
        return;
      }
      case "game.ended": {
        const payload = message.payload as GameEndedEvent;
        setSnapshot((current) => {
          if (!current || !current.game) {
            return current;
          }

          return {
            ...current,
            game: {
              ...current.game,
              phase: "ended",
              settlement: {
                winner_side: payload.winner_side,
                final_multiplier: payload.final_multiplier,
                base_score: current.room.base_score,
                players: payload.settlements.map((item) => ({
                  user_id: item.user_id,
                  seat_index: item.seat_index,
                  role: item.role,
                  score_delta: item.score_delta,
                  is_winner: item.score_delta > 0,
                })),
              },
            },
          };
        });
        return;
      }
      default:
        return;
    }
  }, []);

  const connectRoom = useCallback(
    (nextRoomID: string) => {
      if (!auth.accessToken) {
        return;
      }

      const wsURL = buildRoomWSURL(nextRoomID, auth.accessToken);
      clientRef.current?.disconnect();
      connectionVersionRef.current += 1;
      const connectionVersion = connectionVersionRef.current;
      setRoomID(nextRoomID);
      setSnapshot(null);
      setSelectedCards([]);
      setErrorMessage(null);
      setConnectionState("connecting");

      const client = new RoomSocketClient({
        url: wsURL,
        onOpen: () => {
          if (connectionVersion !== connectionVersionRef.current) {
            return;
          }
          setConnectionState("connected");
          setErrorMessage(null);
        },
        onClose: (state) => {
          if (connectionVersion !== connectionVersionRef.current) {
            return;
          }
          setConnectionState(state);
        },
        onMessage: handleServerMessage,
        onError: (message) => {
          if (connectionVersion !== connectionVersionRef.current) {
            return;
          }
          setErrorMessage(message);
        },
      });

      clientRef.current = client;
      client.connect();
    },
    [auth.accessToken, handleServerMessage],
  );

  const disconnectRoom = useCallback(() => {
    connectionVersionRef.current += 1;
    clientRef.current?.disconnect();
    clientRef.current = null;
    setRoomID(null);
    setSnapshot(null);
    setSelectedCards([]);
    setErrorMessage(null);
    setConnectionState("disconnected");
  }, []);

  useEffect(() => {
    return () => {
      clientRef.current?.disconnect();
      clientRef.current = null;
    };
  }, []);

  const toggleCard = useCallback((card: string) => {
    setSelectedCards((current) => {
      if (current.includes(card)) {
        return current.filter((item) => item !== card);
      }
      return [...current, card];
    });
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedCards([]);
  }, []);

  const sendReady = useCallback((ready: boolean) => {
    sendMessage("room.ready", { ready });
  }, [sendMessage]);

  const sendBid = useCallback((score: 0 | 1 | 2 | 3) => {
    sendMessage("game.bid", { score });
  }, [sendMessage]);

  const sendPlay = useCallback(() => {
    if (!selectedCards.length) {
      return;
    }

    sendMessage("game.play_cards", { cards: selectedCards });
  }, [selectedCards, sendMessage]);

  const sendPass = useCallback(() => {
    sendMessage("game.pass", {});
  }, [sendMessage]);

  const value = useMemo<RoomContextValue>(
    () => ({
      snapshot,
      players: snapshot?.players ?? [],
      me: snapshot?.me ?? null,
      game: snapshot?.game ?? null,
      connectionState,
      errorMessage,
      selectedCards,
      roomID,
      connectRoom,
      disconnectRoom,
      toggleCard,
      clearSelection,
      sendReady,
      sendBid,
      sendPlay,
      sendPass,
    }),
    [
      clearSelection,
      connectRoom,
      connectionState,
      disconnectRoom,
      errorMessage,
      roomID,
      selectedCards,
      sendBid,
      sendPass,
      sendPlay,
      sendReady,
      snapshot,
      toggleCard,
    ],
  );

  return <RoomContext.Provider value={value}>{children}</RoomContext.Provider>;
}

export function useRoom() {
  const context = useContext(RoomContext);
  if (!context) {
    throw new Error("useRoom must be used within RoomProvider");
  }
  return context;
}
