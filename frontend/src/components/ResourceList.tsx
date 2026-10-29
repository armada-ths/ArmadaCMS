import { List, ListProps, SearchInput } from "react-admin";

/** Shared list shell so every resource has the same server-side search. */
export const ResourceList = ({ filters = [], ...props }: ListProps) => (
  <List
    {...props}
    filters={[
      <SearchInput key="q" source="q" alwaysOn />,
      ...(Array.isArray(filters) ? filters : [filters]),
    ]}
  />
);
