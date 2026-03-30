import "server-only";

import { headers } from "next/headers";

import {
  API_BASE,
  ApiError,
  type ApiErrorShape,
  type CurrentUser,
} from "@/lib/api";

function apiUrl(path: string) {
  return `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
}

async function parseJson<T>(response: Response): Promise<T> {
  if (response.status === 204) {
    return null as T;
  }
  return (await response.json()) as T;
}

export async function serverFetchJson<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const requestHeaders = new Headers(init?.headers);
  const headerStore = await headers();
  const cookieHeader = headerStore.get("cookie");

  if (cookieHeader) {
    requestHeaders.set("cookie", cookieHeader);
  }

  if (init?.body && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const response = await fetch(apiUrl(path), {
    ...init,
    cache: "no-store",
    headers: requestHeaders,
  });

  if (!response.ok) {
    const payload = await parseJson<ApiErrorShape>(response);
    throw new ApiError(response.status, payload);
  }

  return parseJson<T>(response);
}

export async function serverFetchJsonSafe<T>(
  path: string,
  init?: RequestInit,
): Promise<T | null> {
  try {
    return await serverFetchJson<T>(path, init);
  } catch (error) {
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403 || error.status === 404)
    ) {
      return null;
    }
    throw error;
  }
}

export async function getCurrentUserSafe() {
  return serverFetchJsonSafe<CurrentUser>("/auth/me");
}
