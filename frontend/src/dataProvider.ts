import {
  createMultipartFormData,
  isMultipartResource,
} from "./utils/multipartFormData";
import simpleRestDataProvider from "ra-data-simple-rest";
import { DataProvider, fetchUtils, HttpError } from "react-admin";
import globalApi from "./context/globalApi";

const endpoint = globalApi();

const getAccessToken = (): string | null => localStorage.getItem("accessToken");

export type FetchJsonResponse = {
  status: number;
  headers: Headers;
  body: string;
  json: unknown;
};

/** Fetch wrapper with Authorization header */
export const httpClient: (
  url: string,
  options?: fetchUtils.Options,
) => Promise<FetchJsonResponse> = (url, options = {}) => {
  const token = getAccessToken();

  const headers = new Headers(
    options.headers || { Accept: "application/json" },
  );
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  options.headers = headers;

  return fetchUtils.fetchJson(url, options);
};

const baseDataProvider = simpleRestDataProvider(endpoint, httpClient);

/** Upload helper */
const uploadFormData = (
  url: string,
  method: "POST" | "PUT",
  formData: FormData,
) => {
  const token = getAccessToken() || "";
  return fetchUtils
    .fetchJson(url, {
      method,
      body: formData,
      credentials: "include",
      headers: new Headers({
        Authorization: `Bearer ${token}`, // ✅ only auth header — no content-type override
      }),
    })
    .then(({ json }) => ({ data: json }));
};

const buildMultipartFormDataOrHttpError = (
  resource: string,
  data: Record<string, unknown>,
) => {
  try {
    return createMultipartFormData(resource, data);
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Unsupported file format.";
    throw new HttpError(message, 400);
  }
};

/** Main data provider */
export const dataProvider: DataProvider = {
  ...baseDataProvider,

  getOne: (resource, params) => {
    return baseDataProvider.getOne(resource, params).then((result) => {
      if (
        resource === "timeline-entries" &&
        result.data.imageUrl &&
        typeof result.data.imageUrl === "string"
      ) {
        return {
          ...result,
          data: { ...result.data, imageUrl: { src: result.data.imageUrl } },
        };
      }
      return result;
    });
  },

  create: (resource, params) => {
    if (isMultipartResource(resource)) {
      try {
        const formData = buildMultipartFormDataOrHttpError(
          resource,
          params.data,
        );
        return uploadFormData(`${endpoint}/${resource}`, "POST", formData);
      } catch (error) {
        return Promise.reject(error);
      }
    }
    return baseDataProvider.create(resource, params);
  },

  update: (resource, params) => {
    if (isMultipartResource(resource)) {
      try {
        const formData = buildMultipartFormDataOrHttpError(
          resource,
          params.data,
        );
        return uploadFormData(
          `${endpoint}/${resource}/${params.id}`,
          "PUT",
          formData,
        );
      } catch (error) {
        return Promise.reject(error);
      }
    }
    return baseDataProvider.update(resource, params);
  },

  delete: (resource, params) => {
    return baseDataProvider.delete(resource, params);
  },
};
