import simpleRestDataProvider from "ra-data-simple-rest";
import {
  CreateParams,
  UpdateParams,
  DataProvider,
  fetchUtils,
} from "react-admin";

const endpoint = import.meta.env.VITE_JSON_SERVER_URL;
const baseDataProvider = simpleRestDataProvider(endpoint);

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

export const dataProvider: DataProvider = {
  ...baseDataProvider,
  create: (resource, params) => {
    if (resource === "profiles") {
      const formData = createPostFormData(params);
      return fetchUtils
        .fetchJson(`${endpoint}/${resource}`, {
          method: "POST",
          body: formData,
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
        })
        .then(({ json }) => ({ data: json }));
    }
    return baseDataProvider.update(resource, params);
  },
};
