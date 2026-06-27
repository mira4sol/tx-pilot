const env = import.meta.env;

function stripTrailingSlash(url: string): string {
  return url.replace(/\/$/, "");
}

function toWebSocketUrl(httpUrl: string): string {
  const base = stripTrailingSlash(httpUrl);
  return `${base.replace(/^http:/i, "ws:").replace(/^https:/i, "wss:")}/v1/ws`;
}

/** Base URL for REST API calls. Empty string uses same origin (Vite dev proxy). */
export const API_BASE = stripTrailingSlash(
  (env.VITE_API_BASE as string | undefined) ??
    (env.VITE_API_URL as string | undefined) ??
    "",
);

/** WebSocket URL for live dashboard updates. */
export function wsUrl(): string {
  const explicit = env.VITE_WS_URL as string | undefined;
  if (explicit) {
    return explicit;
  }
  if (API_BASE) {
    return toWebSocketUrl(API_BASE);
  }
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/v1/ws`;
}
