import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import { APIError } from "../api/http";
import { fetchLobbySummary, fetchRoomList, joinRoom, quickStart } from "../api/lobby";
import { useAuth } from "../auth/AuthContext";
import type {
  LobbySummaryResponse,
  RoomAccessResponse,
  RoomListItem,
} from "../types/api";

type LobbyStatusFilter = "all" | "waiting" | "playing";

interface LobbyContextValue {
  summary: LobbySummaryResponse | null;
  rooms: RoomListItem[];
  totalRooms: number;
  isLoading: boolean;
  isRefreshing: boolean;
  isQuickStarting: boolean;
  joiningRoomID: string | null;
  selectedBaseScore: number;
  selectedStatus: LobbyStatusFilter;
  errorMessage: string | null;
  setSelectedBaseScore: (baseScore: number) => void;
  setSelectedStatus: (status: LobbyStatusFilter) => void;
  refreshLobby: () => Promise<void>;
  quickStartGame: () => Promise<RoomAccessResponse>;
  joinExistingRoom: (roomID: string) => Promise<RoomAccessResponse>;
}

const LobbyContext = createContext<LobbyContextValue | null>(null);

function normalizeErrorMessage(error: unknown) {
  if (error instanceof APIError) {
    switch (error.code) {
      case "room_not_found":
        return "房间不存在，列表已为你刷新。";
      case "already_in_room":
        return "你已经在一个活跃房间中。";
      case "seat_unavailable":
        return "该房间座位已被占用。";
      case "game_already_started":
        return "该房间已经开局，暂时不能加入。";
      default:
        return error.message;
    }
  }

  return "大厅请求失败，请稍后重试";
}

function deriveDefaultBaseScore(summary: LobbySummaryResponse | null) {
  if (summary?.modes.length) {
    return summary.modes[0].base_score;
  }
  return 1;
}

function mapStatusFilter(status: LobbyStatusFilter) {
  if (status === "all") {
    return undefined;
  }
  return status;
}

export function LobbyProvider({ children }: PropsWithChildren) {
  const auth = useAuth();
  const [summary, setSummary] = useState<LobbySummaryResponse | null>(null);
  const [rooms, setRooms] = useState<RoomListItem[]>([]);
  const [totalRooms, setTotalRooms] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isQuickStarting, setIsQuickStarting] = useState(false);
  const [joiningRoomID, setJoiningRoomID] = useState<string | null>(null);
  const [selectedBaseScore, setSelectedBaseScore] = useState(1);
  const [selectedStatus, setSelectedStatus] = useState<LobbyStatusFilter>("all");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const refreshLobby = useCallback(async () => {
    if (!auth.accessToken) {
      return;
    }

    setErrorMessage(null);
    setIsRefreshing(true);
    try {
      const [nextSummary, roomList] = await Promise.all([
        fetchLobbySummary(auth.accessToken),
        fetchRoomList(auth.accessToken, {
          mode: "classic",
          status: mapStatusFilter(selectedStatus),
          page: 1,
          page_size: 20,
        }),
      ]);

      setSummary(nextSummary);
      setRooms(roomList.items);
      setTotalRooms(roomList.total);
      setSelectedBaseScore((current) => {
        const hasCurrent = nextSummary.modes.some((item) => item.base_score === current);
        return hasCurrent ? current : deriveDefaultBaseScore(nextSummary);
      });
    } catch (error) {
      setErrorMessage(normalizeErrorMessage(error));
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, [auth.accessToken, selectedStatus]);

  useEffect(() => {
    void refreshLobby();
  }, [refreshLobby]);

  const quickStartGame = useCallback(async () => {
    if (!auth.accessToken) {
      throw new Error("missing access token");
    }

    setErrorMessage(null);
    setIsQuickStarting(true);
    try {
      return await quickStart(auth.accessToken, selectedBaseScore);
    } catch (error) {
      setErrorMessage(normalizeErrorMessage(error));
      throw error;
    } finally {
      setIsQuickStarting(false);
    }
  }, [auth.accessToken, selectedBaseScore]);

  const joinExistingRoom = useCallback(
    async (roomID: string) => {
      if (!auth.accessToken) {
        throw new Error("missing access token");
      }

      setErrorMessage(null);
      setJoiningRoomID(roomID);
      try {
        return await joinRoom(auth.accessToken, roomID);
      } catch (error) {
        setErrorMessage(normalizeErrorMessage(error));
        await refreshLobby();
        throw error;
      } finally {
        setJoiningRoomID(null);
      }
    },
    [auth.accessToken, refreshLobby],
  );

  const value = useMemo<LobbyContextValue>(
    () => ({
      summary,
      rooms,
      totalRooms,
      isLoading,
      isRefreshing,
      isQuickStarting,
      joiningRoomID,
      selectedBaseScore,
      selectedStatus,
      errorMessage,
      setSelectedBaseScore,
      setSelectedStatus,
      refreshLobby,
      quickStartGame,
      joinExistingRoom,
    }),
    [
      errorMessage,
      isLoading,
      isQuickStarting,
      isRefreshing,
      joiningRoomID,
      quickStartGame,
      refreshLobby,
      rooms,
      selectedBaseScore,
      selectedStatus,
      summary,
      totalRooms,
      joinExistingRoom,
    ],
  );

  return <LobbyContext.Provider value={value}>{children}</LobbyContext.Provider>;
}

export function useLobby() {
  const context = useContext(LobbyContext);
  if (!context) {
    throw new Error("useLobby must be used within LobbyProvider");
  }
  return context;
}
