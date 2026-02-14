import { AuthProvider } from "react-admin";
import { loginApi } from "./authMethods";

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
  return permissions.some((p) => p === "*" || p === required);
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
    if (status === 401 || status === 403) {
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
    const claims = decodeJwtPayload(token);
    return {
      id: claims.user_id as number,
      fullName: (claims.role as string) ?? "User",
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
