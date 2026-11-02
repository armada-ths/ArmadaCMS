import { fetchUtils, HttpError } from "react-admin";
import { withSession } from "./session";

export const authenticatedFetchJson = (
  url: string,
  options: fetchUtils.Options = {},
) =>
  withSession((token) => {
    const headers = new Headers(
      options.headers || { Accept: "application/json" },
    );
    headers.set("Authorization", `Bearer ${token}`);
    return fetchUtils.fetchJson(url, { ...options, headers });
  });

/** For binary downloads and other callers that need the raw response. */
export const authenticatedFetch = (url: string, options: RequestInit = {}) =>
  withSession(async (token) => {
    const headers = new Headers(options.headers);
    headers.set("Authorization", `Bearer ${token}`);
    const response = await fetch(url, { ...options, headers });
    if (response.status === 401) throw new HttpError("Unauthorized", 401);
    return response;
  });
