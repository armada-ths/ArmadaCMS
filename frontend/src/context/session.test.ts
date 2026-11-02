import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import axios, { AxiosError, AxiosHeaders, type AxiosResponse } from "axios";
import { authProvider } from "./authProvider";
import axiosInstance from "./axiosInstance";
import {
  authenticatedFetch,
  authenticatedFetchJson,
} from "./authenticatedHttpClient";
import {
  clearSession,
  getAccessToken,
  refreshTokens,
  SessionExpiredError,
  storeTokens,
  withSession,
} from "./session";
import { dataProvider } from "../dataProvider";

vi.mock("./globalApi", () => ({
  default: () => "https://cms.example.com/api/v1",
}));

const now = Date.now();
const jwt = (expires: number, label = "test") =>
  `header.${btoa(JSON.stringify({ exp: expires / 1000, permissions: ["*"], label }))}.signature`;
const original = {
  accessToken: jwt(now + 15 * 60000),
  refreshToken: "original-refresh",
};
const renewed = {
  accessToken: jwt(now + 60 * 60000, "renewed"),
  refreshToken: "rotated-refresh",
};
const statusError = (status: number) =>
  new AxiosError("Request failed", undefined, undefined, undefined, {
    status,
  } as AxiosResponse);
let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;
const createRefreshMock = () => vi.spyOn(axios, "post");
let refresh: ReturnType<typeof createRefreshMock>;

beforeEach(() => {
  const storage = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
    removeItem: (key: string) => storage.delete(key),
  });
  storeTokens(original);
  vi.spyOn(Date, "now").mockReturnValue(now);
  refresh = createRefreshMock().mockResolvedValue({ data: renewed });
  fetchMock = vi.fn<typeof fetch>().mockImplementation(
    async () =>
      new Response(JSON.stringify({ id: 1, name: "Test User" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
  );
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const idle = () => vi.mocked(Date.now).mockReturnValue(now + 16 * 60000);

it("renews once after 16 idle minutes for simultaneous auth, identity and data requests", async () => {
  idle();
  const [token, identity, result] = await Promise.all([
    getAccessToken(),
    authProvider.getIdentity!(),
    dataProvider.getOne("customusers", { id: 1 }),
    authProvider.checkAuth({}),
  ]);
  expect(token).toBe(renewed.accessToken);
  expect(identity).toEqual({ id: 1, fullName: "Test" });
  expect(result.data.id).toBe(1);
  expect(refresh).toHaveBeenCalledTimes(1);
  expect(localStorage.getItem("refreshToken")).toBe(renewed.refreshToken);
  for (const [, options] of fetchMock.mock.calls) {
    expect(new Headers(options?.headers).get("Authorization")).toBe(
      `Bearer ${renewed.accessToken}`,
    );
  }
});

it("does not refresh a valid access token", async () => {
  await authenticatedFetchJson("https://cms.example.com/api/v1/me");
  expect(refresh).not.toHaveBeenCalled();
});

it("retries a rejected token only once and reuses renewal for a late 401", async () => {
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  const late = vi.fn(async (token: string) => {
    if (token === original.accessToken) {
      await gate;
      throw { status: 401 };
    }
    return "late success";
  });
  const first = vi.fn(async (token: string) => {
    if (token === original.accessToken) throw { status: 401 };
    return "first success";
  });
  const lateRequest = withSession(late);
  await expect(withSession(first)).resolves.toBe("first success");
  release();
  await expect(lateRequest).resolves.toBe("late success");
  expect(refresh).toHaveBeenCalledTimes(1);
  expect(first).toHaveBeenCalledTimes(2);
  expect(late).toHaveBeenCalledTimes(2);
});

it("stops after a second 401, clears the session and signals React-Admin logout", async () => {
  fetchMock.mockImplementation(
    async () => new Response("Unauthorized", { status: 401 }),
  );
  const error = await authenticatedFetchJson(
    "https://cms.example.com/api/v1/me",
  ).catch((e) => e);
  expect(error).toBeInstanceOf(SessionExpiredError);
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(refresh).toHaveBeenCalledTimes(1);
  expect(localStorage.getItem("refreshToken")).toBeNull();
  await expect(authProvider.checkError(error)).rejects.toBe(error);
});

it("logs out when the refresh token has expired", async () => {
  idle();
  refresh.mockRejectedValue(statusError(401));
  await expect(authProvider.checkAuth({})).rejects.toBeInstanceOf(
    SessionExpiredError,
  );
  expect(localStorage.getItem("accessToken")).toBeNull();
  expect(localStorage.getItem("refreshToken")).toBeNull();
  expect(fetchMock).not.toHaveBeenCalled();
});

describe.each([new Error("offline"), statusError(500), statusError(503)])(
  "temporary refresh failure: %s",
  (error) => {
    it("preserves credentials and can recover on the next attempt", async () => {
      idle();
      refresh.mockRejectedValue(error);
      await expect(authProvider.checkAuth({})).resolves.toBeUndefined();
      await expect(
        authenticatedFetchJson("https://cms.example.com/api/v1/me"),
      ).rejects.toBe(error);
      await expect(authProvider.checkError(error)).resolves.toBeUndefined();
      expect(localStorage.getItem("refreshToken")).toBe(original.refreshToken);
      expect(fetchMock).not.toHaveBeenCalled();
      refresh.mockResolvedValue({ data: renewed });
      await expect(getAccessToken()).resolves.toBe(renewed.accessToken);
    });
  },
);

it("does not log out on permission errors", async () => {
  fetchMock.mockResolvedValue(new Response("Forbidden", { status: 403 }));
  const error = await authenticatedFetchJson(
    "https://cms.example.com/api/v1/me",
  ).catch((e) => e);
  await expect(authProvider.checkError(error)).resolves.toBeUndefined();
  expect(refresh).not.toHaveBeenCalled();
  expect(localStorage.getItem("refreshToken")).toBe(original.refreshToken);
});

it("does not restore tokens when logout happens during renewal", async () => {
  let finish!: (value: { data: typeof renewed }) => void;
  refresh.mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  const pending = refreshTokens();
  clearSession();
  finish({ data: renewed });
  await expect(pending).resolves.toBeNull();
  expect(localStorage.getItem("refreshToken")).toBeNull();
});

it("does not overwrite a newer login when old renewal fails", async () => {
  let fail!: (error: unknown) => void;
  refresh.mockImplementation(
    () =>
      new Promise((_resolve, reject) => {
        fail = reject;
      }),
  );
  const pending = refreshTokens();
  storeTokens(renewed);
  fail(statusError(401));
  await expect(pending).resolves.toEqual(renewed);
  await expect(
    authProvider.checkError(new SessionExpiredError()),
  ).resolves.toBeUndefined();
  expect(localStorage.getItem("refreshToken")).toBe(renewed.refreshToken);
});

it("rechecks storage inside the cross-tab lock", async () => {
  vi.stubGlobal("navigator", {
    locks: {
      request: vi.fn(
        async (_name: string, callback: () => Promise<unknown>) => {
          storeTokens(renewed); // another tab rotated before our lock was granted
          return callback();
        },
      ),
    },
  });
  await expect(refreshTokens()).resolves.toEqual(renewed);
  expect(refresh).not.toHaveBeenCalled();
});

it("renews multipart uploads and preserves their body and content type", async () => {
  idle();
  const form = new FormData();
  form.append("name", "Example");
  await authenticatedFetchJson("https://cms.example.com/api/v1/profiles/1", {
    method: "PUT",
    body: form,
  });
  const [, options] = fetchMock.mock.calls[0];
  expect(options?.body).toBe(form);
  expect(new Headers(options?.headers).has("Content-Type")).toBe(false);
  expect(new Headers(options?.headers).get("Authorization")).toBe(
    `Bearer ${renewed.accessToken}`,
  );
});

it("renews binary downloads and retries a 401 without parsing the body as JSON", async () => {
  fetchMock.mockResolvedValueOnce(
    new Response("Unauthorized", { status: 401 }),
  );
  fetchMock.mockResolvedValueOnce(
    new Response("<svg />", { headers: { "Content-Type": "image/svg+xml" } }),
  );
  const response = await authenticatedFetch(
    "https://cms.example.com/api/v1/photoevents/1/qr",
  );
  expect(await response.text()).toBe("<svg />");
  expect(refresh).toHaveBeenCalledTimes(1);
});

it("uses the shared renewal for axios requests after inactivity", async () => {
  idle();
  const adapter = vi.fn(async (config) => ({
    data: "ok",
    status: 200,
    statusText: "OK",
    headers: new AxiosHeaders(),
    config,
  }));
  const [response] = await Promise.all([
    axiosInstance.get("/eventrofairs", { adapter }),
    getAccessToken(),
  ]);
  expect(response.data).toBe("ok");
  expect(adapter.mock.calls[0][0].headers.Authorization).toBe(
    `Bearer ${renewed.accessToken}`,
  );
  expect(refresh).toHaveBeenCalledTimes(1);
});

it("retries axios 401 only once", async () => {
  const adapter = vi.fn(async (config) => {
    throw new AxiosError("Unauthorized", undefined, config, undefined, {
      status: 401,
      config,
      headers: new AxiosHeaders(),
      statusText: "Unauthorized",
      data: "",
    });
  });
  await expect(
    axiosInstance.get("/eventrofairs", { adapter }),
  ).rejects.toBeInstanceOf(SessionExpiredError);
  expect(adapter).toHaveBeenCalledTimes(2);
  expect(refresh).toHaveBeenCalledTimes(1);
});

it("does not attach or refresh session credentials for a failed login", async () => {
  const adapter = vi.fn(async (config) => {
    throw new AxiosError("Wrong password", undefined, config, undefined, {
      status: 401,
      config,
      headers: new AxiosHeaders(),
      statusText: "Unauthorized",
      data: "",
    });
  });
  await expect(
    axiosInstance.post("/login", {}, { adapter }),
  ).rejects.toBeInstanceOf(AxiosError);
  expect(adapter.mock.calls[0][0].headers.Authorization).toBeUndefined();
  expect(refresh).not.toHaveBeenCalled();
  expect(localStorage.getItem("refreshToken")).toBe(original.refreshToken);
});
