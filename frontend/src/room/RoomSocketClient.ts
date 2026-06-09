import type { RoomClientMessage, RoomClientMessageType, RoomConnectionState, RoomServerMessage } from "./roomTypes";

interface RoomSocketClientOptions {
  url: string;
  onOpen: () => void;
  onClose: (state: RoomConnectionState) => void;
  onMessage: (message: RoomServerMessage) => void;
  onError: (message: string) => void;
}

const heartbeatIntervalMS = 15_000;
const reconnectDelayMS = 1_500;

export class RoomSocketClient {
  private readonly options: RoomSocketClientOptions;
  private socket: WebSocket | null = null;
  private messageSeq = 0;
  private heartbeatTimer: number | null = null;
  private reconnectTimer: number | null = null;
  private manuallyClosed = false;

  constructor(options: RoomSocketClientOptions) {
    this.options = options;
  }

  connect() {
    this.manuallyClosed = false;
    this.clearReconnectTimer();
    this.openSocket();
  }

  disconnect() {
    this.manuallyClosed = true;
    this.clearHeartbeatTimer();
    this.clearReconnectTimer();
    this.socket?.close();
    this.socket = null;
  }

  send<TPayload>(type: RoomClientMessageType, payload: TPayload) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error("room websocket not connected");
    }

    this.messageSeq += 1;
    const requestID = `req_${Date.now()}_${this.messageSeq}`;
    const message: RoomClientMessage<TPayload> = {
      type,
      request_id: requestID,
      seq: this.messageSeq,
      payload,
    };

    this.socket.send(JSON.stringify(message));
    return requestID;
  }

  private openSocket() {
    const socket = new WebSocket(this.options.url);
    this.socket = socket;

    socket.addEventListener("open", () => {
      this.options.onOpen();
      this.startHeartbeat();
    });

    socket.addEventListener("message", (event) => {
      try {
        const message = JSON.parse(String(event.data)) as RoomServerMessage;
        this.options.onMessage(message);
      } catch {
        this.options.onError("房间消息解析失败");
      }
    });

    socket.addEventListener("error", () => {
      this.options.onError("房间连接异常");
    });

    socket.addEventListener("close", () => {
      this.clearHeartbeatTimer();
      this.socket = null;

      if (this.manuallyClosed) {
        this.options.onClose("disconnected");
        return;
      }

      this.options.onClose("reconnecting");
      this.reconnectTimer = window.setTimeout(() => {
        this.openSocket();
      }, reconnectDelayMS);
    });
  }

  private startHeartbeat() {
    this.clearHeartbeatTimer();
    this.heartbeatTimer = window.setInterval(() => {
      if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
        return;
      }

      try {
        this.send("ping", { client_time: new Date().toISOString() });
      } catch {
        this.options.onError("心跳发送失败");
      }
    }, heartbeatIntervalMS);
  }

  private clearHeartbeatTimer() {
    if (this.heartbeatTimer !== null) {
      window.clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private clearReconnectTimer() {
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
