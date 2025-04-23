import { AuthProvider } from "react-admin";
import { loginApi } from "./authMethods";

export const authProvider: AuthProvider = {
  async login({ username, password }) {
    // if (username !== "john" || password !== "123") {
    //   throw new Error("Login failed");
    // }
    // localStorage.setItem("username", username);
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
      localStorage.removeItem("username");
      throw new Error("Session expired");
    }
    // other error codes (404, 500, etc): no need to log out
  },
  async checkAuth() {
    if (
      !localStorage.getItem("accessToken") ||
      !localStorage.getItem("refreshToken")
    ) {
      throw new Error("Not authenticated");
    }
  },
  async logout() {
    localStorage.removeItem("username");
  },
  async getIdentity() {
    const username = localStorage.getItem("username");
    if (!username) {
      throw new Error("No identity found");
    }

    return { id: username, fullName: username };
  },
};
