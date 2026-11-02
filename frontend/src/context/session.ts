import axios from "axios";
import type { Tokens } from "../models/user";
import globalApi from "./globalApi";

export class SessionExpiredError extends Error {
  readonly status = 401;
  constructor() {
    super("Session expired");
  }
}

export function clearSession() {
  localStorage.removeItem("accessToken");
  localStorage.removeItem("refreshToken");
}

export function storeTokens(tokens: Tokens) {
  localStorage.setItem("accessToken", tokens.accessToken);
  localStorage.setItem("refreshToken", tokens.refreshToken);
}

export function decodeJwtPayload(token: string): Record<string, unknown> {
  try {
    const base64 = token.split(".")[1];
    return JSON.parse(atob(base64.replace(/-/g, "+").replace(/_/g, "/")));
  } catch {
    return {};
  }
}

function storedTokens(): Tokens | null {
  const accessToken = localStorage.getItem("accessToken");
  const refreshToken = localStorage.getItem("refreshToken");
  return accessToken && refreshToken ? { accessToken, refreshToken } : null;
}

function statusOf(error: unknown): number | undefined {
  if (axios.isAxiosError(error)) return error.response?.status;
  if (error && typeof error === "object" && "status" in error) {
    return error.status as number | undefined;
  }
}

let pendingRefresh: Promise<Tokens | null> | undefined;

export function refreshTokens(): Promise<Tokens | null> {
  if (pendingRefresh) return pendingRefresh;
  const refreshToken = localStorage.getItem("refreshToken");
  if (!refreshToken) return Promise.resolve(null);

  const rotate = async (): Promise<Tokens | null> => {
    // Another tab may have rotated the token while we waited for the lock.
    if (localStorage.getItem("refreshToken") !== refreshToken) {
      return storedTokens();
    }
    try {
      const { data } = await axios.post<Tokens>(
        `${globalApi()}/refreshAccessToken`,
        undefined,
        {
          headers: { "X-RefreshAuthorization": `Bearer ${refreshToken}` },
          timeout: 30000,
        },
      );
      // Do not resurrect a logged-out session or overwrite a newer login.
      if (localStorage.getItem("refreshToken") !== refreshToken) {
        return storedTokens();
      }
      if (!data.accessToken || !data.refreshToken) {
        throw new Error("Invalid token refresh response");
      }
      storeTokens(data);
      return data;
    } catch (error) {
      if (localStorage.getItem("refreshToken") !== refreshToken) {
        return storedTokens();
      }
      if (statusOf(error) === 401) {
        clearSession();
        return null;
      }
      // Offline, timeout and server errors do not invalidate credentials.
      throw error;
    }
  };

  // Web Locks serialize rotation across same-origin tabs; the shared promise
  // also coordinates callers in browsers without Web Locks.
  pendingRefresh = (
    typeof navigator !== "undefined" && navigator.locks
      ? navigator.locks.request("armadacms-session-refresh", rotate)
      : rotate()
  ).finally(() => {
    pendingRefresh = undefined;
  });
  return pendingRefresh;
}

export async function getAccessToken(): Promise<string> {
  const tokens = storedTokens();
  if (!tokens) throw new SessionExpiredError();
  const exp = decodeJwtPayload(tokens.accessToken).exp;
  // Refresh a little early so the token remains valid during the request.
  if (typeof exp === "number" && exp * 1000 > Date.now() + 30000) {
    return tokens.accessToken;
  }
  const refreshed = await refreshTokens();
  if (!refreshed) throw new SessionExpiredError();
  return refreshed.accessToken;
}

export async function recoverAccessToken(
  rejectedToken: string,
): Promise<string> {
  const current = storedTokens();
  if (!current) throw new SessionExpiredError();
  // A late 401 from an old request must reuse the already refreshed token.
  if (current.accessToken !== rejectedToken) return getAccessToken();
  const refreshed = await refreshTokens();
  if (!refreshed) throw new SessionExpiredError();
  return refreshed.accessToken;
}

export function rejectAccessToken(token: string): never {
  if (localStorage.getItem("accessToken") === token) clearSession();
  throw new SessionExpiredError();
}

export async function withSession<T>(
  request: (token: string) => Promise<T>,
): Promise<T> {
  let token = await getAccessToken();
  try {
    return await request(token);
  } catch (error) {
    if (statusOf(error) !== 401) throw error;
  }
  token = await recoverAccessToken(token);
  try {
    return await request(token);
  } catch (error) {
    if (statusOf(error) === 401) rejectAccessToken(token);
    throw error;
  }
}
