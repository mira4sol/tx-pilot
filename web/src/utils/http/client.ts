import { API_BASE } from "./config";

export class HttpError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body?: unknown,
  ) {
    super(message);
    this.name = "HttpError";
  }
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  params?: Record<string, string | number | undefined>;
}

function buildUrl(path: string, params?: RequestOptions["params"]): string {
  const base = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
  if (!params) return base;
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined) {
      search.set(key, String(value));
    }
  }
  const qs = search.toString();
  return qs ? `${base}?${qs}` : base;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, params, headers, ...init } = options;
  const response = await fetch(buildUrl(path, params), {
    ...init,
    headers: {
      Accept: "application/json",
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await response.text();
  let parsed: unknown = undefined;
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = text;
    }
  }

  if (!response.ok) {
    const message =
      typeof parsed === "object" &&
      parsed !== null &&
      "error" in parsed &&
      typeof (parsed as { error: unknown }).error === "string"
        ? (parsed as { error: string }).error
        : response.statusText || "Request failed";
    throw new HttpError(message, response.status, parsed);
  }

  return parsed as T;
}

export function get<T>(path: string, params?: RequestOptions["params"]): Promise<T> {
  return request<T>(path, { method: "GET", params });
}

export function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: "POST", body });
}
