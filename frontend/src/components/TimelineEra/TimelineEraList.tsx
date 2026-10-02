import {
  Datagrid,
  DeleteButton,
  EditButton,
  List,
  ListProps,
  NumberField,
  TextField,
} from "react-admin";

export const TimelineEraList = (props: ListProps) => (
  <List {...props} sort={{ field: "sortOrder", order: "ASC" }}>
    <Datagrid>
      <TextField source="title" label="Title" sortable={false} />
      <NumberField source="sortOrder" label="Sort order" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
