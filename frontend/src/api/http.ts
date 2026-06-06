import { env } from "../config/env";
import type { APIErrorPayload, APIResponse } from "../types/api";

export class APIError extends Error {
  code: string;
  status: number;
  requestId?: string;

  constructor(payload: APIErrorPayload) {
    super(payload.message);
    this.name = "APIError";
    this.code = payload.code;
    this.status = payload.status;
    this.requestId = payload.requestId;
  }
}

interface RequestOptions extends RequestInit {
  token?: string;
}

export async function requestJSON<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");

  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }

  const response = await fetch(`${env.apiBaseURL}${path}`, {
    ...options,
    headers,
  });

  const payload = (await response.json()) as APIResponse<T>;
  if (!response.ok || payload.code !== "ok") {
    throw new APIError({
      code: payload.code,
      message: payload.message,
      requestId: payload.request_id,
      status: response.status,
    });
  }

  return payload.data;
}
