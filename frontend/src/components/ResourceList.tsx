import {
  DateInput,
  DateTimeInput,
  List,
  ListProps,
  NullableBooleanInput,
  NumberInput,
  SearchInput,
  SelectInput,
  TextInput,
  useResourceContext,
} from "react-admin";
import { RESOURCE_FILTERS, type ResourceFilter } from "./resourceFilters";
import { toLocalInputValue, toUTCISOString } from "../utils/dateTimeHelpers";

const columnFilter = ({ source, label, type, choices }: ResourceFilter) => {
  const props = { source, label };
  switch (type) {
    case "number":
      return <NumberInput key={source} {...props} />;
    case "boolean":
      return <NullableBooleanInput key={source} {...props} />;
    case "date":
      return <DateInput key={source} {...props} />;
    case "datetime":
      return (
        <DateTimeInput
          key={source}
          {...props}
          format={toLocalInputValue}
          parse={toUTCISOString}
        />
      );
    case "select":
      return <SelectInput key={source} {...props} choices={choices} />;
    default:
      return <TextInput key={source} {...props} />;
  }
};

/** Shared search and exact column filters for every resource list. */
export const ResourceList = ({ filters = [], ...props }: ListProps) => {
  const resource = useResourceContext(props);
  const definitions =
    RESOURCE_FILTERS[resource as keyof typeof RESOURCE_FILTERS] ?? [];
  return (
    <List
      {...props}
      filters={[
        <SearchInput key="q" source="q" alwaysOn />,
        ...definitions.map(columnFilter),
        ...(Array.isArray(filters) ? filters : [filters]),
      ]}
    />
  );
};
