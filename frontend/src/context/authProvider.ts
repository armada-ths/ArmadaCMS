import { AuthProvider } from "react-admin";
import { loginApi } from "./authMethods";
import {
  clearSession,
  decodeJwtPayload,
  getAccessToken,
  SessionExpiredError,
  storeTokens,
} from "./session";
import { authenticatedFetchJson } from "./authenticatedHttpClient";
import globalApi from "./globalApi";
import { hasPerm } from "../utils/permissions";

function getPermissionsFromToken(): string[] {
  const token = localStorage.getItem("accessToken");
  if (!token) return [];
  const claims = decodeJwtPayload(token);
  return (claims.permissions as string[]) ?? [];
}

export const authProvider: AuthProvider = {
  async login({ username, password }) {
    try {
      const tokens = await loginApi({ username, password });

      storeTokens(tokens);

      return Promise.resolve();
    } catch (error) {
      return Promise.reject(error);
    }
  },

  async checkError(error) {
    // The HTTP clients exhaust renewal before reporting an expired session.
    // Credentials are cleared there, guarded against late failures from old requests.
    if (
      error instanceof SessionExpiredError &&
      !localStorage.getItem("refreshToken")
    ) {
      throw error;
    }
  },

  async checkAuth() {
    try {
      await getAccessToken();
    } catch (error) {
      if (error instanceof SessionExpiredError) throw error;
      // React-Admin treats a rejected checkAuth as logout. A transient refresh
      // failure should instead leave the session available for the next attempt.
    }
  },

  async logout() {
    clearSession();
  },

  async getIdentity() {
    const { json } = await authenticatedFetchJson(`${globalApi()}/me`);
    const user = json as { id: number; name?: string; username?: string };
    const firstName = (user.name as string)?.trim().split(/\s+/)[0];
    const displayName = firstName || (user.username as string) || "User";
    return {
      id: user.id as number,
      fullName: displayName,
    };
  },

  async getPermissions() {
    return getPermissionsFromToken();
  },

  async canAccess({ action, resource }: { action: string; resource: string }) {
    const permissions = getPermissionsFromToken();
    // React-Admin uses "list" and "show" internally; our permission system uses "view"
    const mappedAction =
      action === "list" || action === "show" ? "view" : action;
    const group =
      resource === "lounge-items" || resource === "lounge-categories"
        ? "lounge-catalogue"
        : resource;
    const permissionKey = `${group}.${mappedAction}`;
    return hasPerm(permissions, permissionKey);
  },
};
