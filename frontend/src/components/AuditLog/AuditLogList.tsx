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
} from "react-admin";

const auditLogFilters = [
  <TextInput key="q" label="Search" source="q" alwaysOn />,
  <SelectInput
    key="action"
    source="action"
    choices={[
      { id: "create", name: "Create" },
      { id: "update", name: "Update" },
      { id: "delete", name: "Delete" },
    ]}
  />,
  <TextInput
    key="resource_type"
    source="resource_type"
    label="Resource type"
  />,
  <TextInput key="actor_username" source="actor_username" label="Username" />,
];

const ListActions = () => (
  <TopToolbar>
    <FilterButton />
  </TopToolbar>
);

export const AuditLogList = (props: ListProps) => (
  <List
    {...props}
    sort={{ field: "created_at", order: "DESC" }}
    filters={auditLogFilters}
    actions={<ListActions />}
  >
    <Datagrid bulkActionButtons={false} rowClick="show">
      <DateField source="created_at" label="Time" showTime />
      <TextField source="action" />
      <TextField source="resource_type" label="Resource" />
      <TextField source="resource_id" label="Resource ID" />
      <TextField source="actor_username" label="User" />
      <TextField source="http_method" label="Method" />
      <TextField source="request_path" label="Path" />
      <ShowButton />
    </Datagrid>
  </List>
);
