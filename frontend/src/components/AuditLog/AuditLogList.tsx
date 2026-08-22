import { FormControlLabel, Switch } from "@mui/material";
import {
  List,
  Datagrid,
  TextField,
  DateField,
  ShowButton,
  TextInput,
  SelectInput,
  FilterButton,
  TopToolbar,
  ListProps,
  useListContext,
} from "react-admin";
import { AuditLogChildren } from "./AuditLogChildren";

const auditLogFilters = [
  <TextInput key="q" label="Search" source="q" alwaysOn />,
  <SelectInput
    key="action"
    source="action"
    choices={[
      { id: "create", name: "Create" },
      { id: "update", name: "Update" },
      { id: "delete", name: "Delete" },
      { id: "sync", name: "Sync" },
    ]}
  />,
  <TextInput
    key="resource_type"
    source="resource_type"
    label="Resource type"
  />,
  <TextInput key="actor_username" source="actor_username" label="Username" />,
];

const ListActions = () => {
  const { displayedFilters, filterValues, setFilters } = useListContext();
  const showLoginActivity = filterValues.include_auth === "true";

  const handleToggle = (checked: boolean) => {
    const nextFilters = { ...filterValues };
    if (checked) {
      nextFilters.include_auth = "true";
    } else {
      delete nextFilters.include_auth;
    }
    setFilters(nextFilters, displayedFilters);
  };

  return (
    <TopToolbar>
      <FormControlLabel
        control={
          <Switch
            checked={showLoginActivity}
            onChange={(_, checked) => handleToggle(checked)}
          />
        }
        label="Show login activity"
      />
      <FilterButton />
    </TopToolbar>
  );
};

export const AuditLogList = (props: ListProps) => (
  <List
    {...props}
    sort={{ field: "created_at", order: "DESC" }}
    filters={auditLogFilters}
    actions={<ListActions />}
  >
    <Datagrid
      bulkActionButtons={false}
      rowClick="show"
      expand={<AuditLogChildren />}
      isRowExpandable={(record) => Number(record.child_count ?? 0) > 0}
    >
      <DateField source="created_at" label="Time" showTime />
      <TextField source="action" />
      <TextField source="resource_type" label="Resource" />
      <TextField source="resource_id" label="Resource ID" />
      <TextField source="actor_username" label="User" />
      <TextField source="http_method" label="Method" />
      <TextField source="request_path" label="Path" />
      <TextField source="child_count" label="Related" />
      <ShowButton />
    </Datagrid>
  </List>
);
