import { AuthProvider } from "react-admin";
import { loginApi } from "./authMethods";

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
    // optional: decode JWT to extract identity info
    const token = localStorage.getItem("accessToken");
    if (!token) {
      throw new Error("No identity found");
    }
    return { id: "me", fullName: "Authenticated User" };
  },
};
