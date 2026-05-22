import { AuthProvider } from "react-admin";
import { loginApi } from "./authMethods";
import { refreshTokens } from "./axiosInstance";
import globalApi from "./globalApi";

/**
 * Decode a JWT payload without verification (browser-side).
 * We only need this to read claims; the server validates the signature.
 */
function decodeJwtPayload(token: string): Record<string, unknown> {
  try {
    const base64 = token.split(".")[1];
    const json = atob(base64.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(json);
  } catch {
    return {};
  }
}

function getPermissionsFromToken(): string[] {
  const token = localStorage.getItem("accessToken");
  if (!token) return [];
  const claims = decodeJwtPayload(token);
  return (claims.permissions as string[]) ?? [];
}

function hasPermission(permissions: string[], required: string): boolean {
  const dotIndex = required.indexOf(".");
  const hasResourceAction = dotIndex !== -1;
  const reqResource = hasResourceAction
    ? required.slice(0, dotIndex)
    : required;
  const reqAction = hasResourceAction ? required.slice(dotIndex + 1) : "";

  return permissions.some((p) => {
    if (p === "*" || p === required) return true;
    if (!hasResourceAction) return false;
    const pDot = p.indexOf(".");
    if (pDot === -1) return false;
    const pResource = p.slice(0, pDot);
    const pAction = p.slice(pDot + 1);
    // "resource.*" grants all actions on that resource
    if (pResource === reqResource && pAction === "*") return true;
    // "*.action" grants that action on all resources
    if (pResource === "*" && pAction === reqAction) return true;
    return false;
  });
}

export const authProvider: AuthProvider = {
  async login({ username, password }) {
    try {
      const tokens = await loginApi({ username, password });

      localStorage.setItem("accessToken", tokens.accessToken);
      localStorage.setItem("refreshToken", tokens.refreshToken);

      return Promise.resolve();
    } catch (error) {
      return Promise.reject(error);
    }
  },

  async checkError(error) {
    const status = error.status;
    if (status === 401) {
      localStorage.removeItem("accessToken");
      localStorage.removeItem("refreshToken");
      throw new Error("Session expired");
    }
    return Promise.resolve();
  },

  async checkAuth() {
    const access = localStorage.getItem("accessToken");
    const refresh = localStorage.getItem("refreshToken");
    if (!access || !refresh) {
      throw new Error("Not authenticated");
    }

    const claims = decodeJwtPayload(access);
    const exp = claims.exp as number | undefined;
    if (exp && exp * 1000 < Date.now()) {
      // Access token expired — attempt silent refresh before forcing logout
      const tokens = await refreshTokens();
      if (!tokens) {
        throw new Error("Session expired");
      }
      localStorage.setItem("accessToken", tokens.accessToken);
      localStorage.setItem("refreshToken", tokens.refreshToken);
    }

    return Promise.resolve();
  },

  async logout() {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    return Promise.resolve();
  },

  async getIdentity() {
    const token = localStorage.getItem("accessToken");
    if (!token) {
      throw new Error("No identity found");
    }
    const response = await fetch(`${globalApi()}/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!response.ok) {
      throw new Error("Failed to fetch identity");
    }
    const user = await response.json();
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
    // Map React-Admin actions to our permission strings
    const permissionKey = `${resource}.${action}`;
    return hasPermission(permissions, permissionKey);
  },
};
