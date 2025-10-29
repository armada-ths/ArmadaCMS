import simpleRestDataProvider from "ra-data-simple-rest";
import {
  CreateParams,
  UpdateParams,
  DataProvider,
  fetchUtils,
} from "react-admin";
import globalApi from "./context/globalApi";

const endpoint = globalApi();

/** Fetch wrapper with Authorization header */
export const httpClient: (
  url: string,
  options?: fetchUtils.Options,
) => Promise<FetchJsonResponse> = (url, options = {}) => {
  const token = localStorage.getItem("accessToken");

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

export type FetchJsonResponse = {
  status: number;
  headers: Headers;
  body: string;
  json: unknown;
};

/** Build FormData for multipart upload (profiles, events, etc.) */
const createMultipartFormData = (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  params: CreateParams<any> | UpdateParams<any>,
) => {
  const formData = new FormData();

  Object.entries(params.data).forEach(([key, value]) => {
    if (value == null) return;

    // Handle React Admin ImageInput
    if (value.rawFile instanceof File) {
      formData.append("file", value.rawFile);
    }

    // Handle logo/image/link fields
    else if (typeof value === "string" && /(photo|image|logo|img)/i.test(key)) {
      // ✅ For exhibitors: logoFreesize, logoSquared, mapImg
      formData.append(key, value);
    } else if (Array.isArray(value)) {
      // 👇 include all arrays (programs, industries, employments) as JSON
      formData.append(key, JSON.stringify(value));
    }
    // Handle scalar values
    else if (typeof value !== "object") {
      formData.append(key, String(value));
    }
  });

  return formData;
};

/** Upload helper */
const uploadFormData = (
  url: string,
  method: "POST" | "PUT",
  formData: FormData,
) => {
  const token = localStorage.getItem("accessToken") || "";
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

/** Main data provider */
export const dataProvider: DataProvider = {
  ...baseDataProvider,

  create: (resource, params) => {
    if (["profiles", "events", "exhibitors"].includes(resource)) {
      const formData = createMultipartFormData(params);
      return uploadFormData(`${endpoint}/${resource}`, "POST", formData);
    }
    return baseDataProvider.create(resource, params);
  },

  update: (resource, params) => {
    if (["profiles", "events", "exhibitors"].includes(resource)) {
      const formData = createMultipartFormData(params);
      return uploadFormData(
        `${endpoint}/${resource}/${params.id}`,
        "PUT",
        formData,
      );
    }
    return baseDataProvider.update(resource, params);
  },
};
