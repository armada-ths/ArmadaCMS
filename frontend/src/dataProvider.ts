import simpleRestProvider from "ra-data-simple-rest";

export const dataProvider = simpleRestProvider(
  import.meta.env.VITE_JSON_SERVER_URL,
);
