import axios from "axios";

import { Tokens } from "../models/user";
import globalApi from "./globalApi";

const axiosInstance = axios.create({
  baseURL: globalApi(),
  headers: { "Content-Type": "application/json" },
});

axiosInstance.interceptors.request.use((config) => {
  const accessToken = localStorage.getItem("accessToken");
  const refreshToken = localStorage.getItem("refreshToken");

  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`;
  if (refreshToken)
    config.headers["X-RefreshAuthorization"] = `Bearer ${refreshToken}`;

  return config;
});

/**
 * Exchange the stored refresh token for a new token pair.
 * Uses a plain axios call (not the intercepted instance) to avoid
 * triggering this interceptor recursively.
 * Returns null and clears storage when the refresh token is invalid/expired.
 */
export const refreshTokens = async (): Promise<Tokens | null> => {
  const refreshToken = localStorage.getItem("refreshToken");
  if (!refreshToken) return null;
  try {
    const response = await axios.post<Tokens>(
      `${globalApi()}/refreshAccessToken`,
      undefined,
      { headers: { "X-RefreshAuthorization": `Bearer ${refreshToken}` } },
    );
    return response.data;
  } catch {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    return null;
  }
};

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    // Only retry once and skip the refresh endpoint itself to avoid loops
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      const tokens = await refreshTokens();
      if (tokens) {
        localStorage.setItem("accessToken", tokens.accessToken);
        localStorage.setItem("refreshToken", tokens.refreshToken);
        originalRequest.headers.Authorization = `Bearer ${tokens.accessToken}`;
        return axiosInstance(originalRequest);
      }
    }
    return Promise.reject(error);
  },
);

export default axiosInstance;
