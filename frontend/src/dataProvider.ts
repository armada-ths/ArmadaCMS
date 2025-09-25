import simpleRestDataProvider from "ra-data-simple-rest";
import {
  CreateParams,
  UpdateParams,
  DataProvider,
  fetchUtils,
} from "react-admin";
import globalApi from "./context/globalApi";

const endpoint = globalApi();
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

type PostParams = {
  id: string;
  title: string;
  content: string;
  name: string;
  team_id: number;
  photoFile: {
    rawFile: File;
    src?: string;
    title?: string;
  };
};
const createPostFormData = (
  params: CreateParams<PostParams> | UpdateParams<PostParams>,
) => {
  const formData = new FormData();

  // Extract and append file
  const file = params.data.photoFile?.rawFile;
  if (file instanceof File) {
    formData.append("file", file);
  }

  // Append all other fields except photoFile
  Object.entries(params.data).forEach(([key, value]) => {
    if (key !== "photoFile" && value !== undefined && value !== null) {
      formData.append(key, String(value));
    }
  });

  return formData;
};
export type FetchJsonResponse = {
  status: number;
  headers: Headers;
  body: string;
  json: unknown;
};

export const dataProvider: DataProvider = {
  ...baseDataProvider,
  create: (resource, params) => {
    if (resource === "profiles") {
      const formData = createPostFormData(params);
      return fetchUtils
        .fetchJson(`${endpoint}/${resource}`, {
          method: "POST",
          body: formData,
          credentials: "include",
        })
        .then(({ json }) => ({ data: json }));
    }
    return baseDataProvider.create(resource, params);
  },
  update: (resource, params) => {
    if (resource === "profiles") {
      const formData = createPostFormData(params);
      return fetchUtils
        .fetchJson(`${endpoint}/${resource}/${params.id}`, {
          method: "PUT",
          body: formData,
          credentials: "include",
        })
        .then(({ json }) => ({ data: json }));
    }
    return baseDataProvider.update(resource, params);
  },
};
