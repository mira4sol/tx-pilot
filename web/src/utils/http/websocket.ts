import { wsUrl } from "./config";
import type { WsEnvelope } from "./types";
import { WS_CHANNELS } from "./types";

export type WsMessageHandler = (message: WsEnvelope) => void;
export type WsStatusHandler = (connected: boolean) => void;

const RECONNECT_MS = 3000;

export class DashboardWebSocket {
  private socket: WebSocket | null = null;
  private handlers = new Set<WsMessageHandler>();
  private statusHandlers = new Set<WsStatusHandler>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;

  connect(channels: readonly string[] = WS_CHANNELS): void {
    this.closed = false;
    this.clearReconnect();
    this.socket?.close();

    const ws = new WebSocket(wsUrl());
    this.socket = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "subscribe", channels: [...channels] }));
      this.emitStatus(true);
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data as string) as WsEnvelope;
        for (const handler of this.handlers) {
          handler(message);
        }
      } catch {
        // Ignore malformed frames.
      }
    };

    ws.onclose = () => {
      this.emitStatus(false);
      if (!this.closed) {
        this.scheduleReconnect(channels);
      }
    };

    ws.onerror = () => {
      ws.close();
    };
  }

  onMessage(handler: WsMessageHandler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  onStatus(handler: WsStatusHandler): () => void {
    this.statusHandlers.add(handler);
    return () => this.statusHandlers.delete(handler);
  }

  disconnect(): void {
    this.closed = true;
    this.clearReconnect();
    this.socket?.close();
    this.socket = null;
    this.emitStatus(false);
  }

  private scheduleReconnect(channels: readonly string[]): void {
    this.clearReconnect();
    this.reconnectTimer = setTimeout(() => this.connect(channels), RECONNECT_MS);
  }

  private clearReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private emitStatus(connected: boolean): void {
    for (const handler of this.statusHandlers) {
      handler(connected);
    }
  }
}
