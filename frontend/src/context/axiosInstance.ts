import axios, { type InternalAxiosRequestConfig } from "axios";
import globalApi from "./globalApi";
import {
  getAccessToken,
  recoverAccessToken,
  rejectAccessToken,
} from "./session";

type SessionRequest = InternalAxiosRequestConfig & { _sessionRetry?: boolean };
const axiosInstance = axios.create({
  baseURL: globalApi(),
  headers: { "Content-Type": "application/json" },
});

const isLogin = (config: InternalAxiosRequestConfig) =>
  config.url?.replace(/^\//, "") === "login";

axiosInstance.interceptors.request.use(async (config) => {
  if (!isLogin(config)) {
    config.headers.Authorization = `Bearer ${await getAccessToken()}`;
  }
  return config;
});

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: unknown) => {
    if (
      !axios.isAxiosError(error) ||
      error.response?.status !== 401 ||
      !error.config
    ) {
      return Promise.reject(error);
    }
    const config: SessionRequest = error.config;
    if (isLogin(config)) return Promise.reject(error);
    const rejected = String(config.headers.Authorization ?? "").replace(
      /^Bearer /,
      "",
    );
    if (config._sessionRetry) rejectAccessToken(rejected);
    config._sessionRetry = true;
    const token = await recoverAccessToken(rejected);
    config.headers.Authorization = `Bearer ${token}`;
    return axiosInstance(config);
  },
);

export default axiosInstance;
