import axios from "axios";
// import { refreshToken } from "./auth"; // Function to get a new token

import { Tokens } from "../models/user";

const axiosInstance = axios.create({
  baseURL: import.meta.env.VITE_JSON_SERVER_URL, // Your API base URL
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

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      const tokens = await refreshToken();
      if (tokens) {
        localStorage.setItem("accessToken", tokens.accessToken);
        localStorage.setItem("refreshToken", tokens.refreshToken);
        error.config.headers.Authorization = `Bearer ${tokens.accessToken}`;
        return axiosInstance(error.config);
      }
    }
    return Promise.reject(error);
  },
);

const refreshToken = async () => {
  try {
    const response = await axiosInstance.get("refreshAccessToken");

    const tokens: Tokens = response.data;

    return tokens;
  } catch (error) {
    console.error(error);
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    return null;
  }
};

export default axiosInstance;
