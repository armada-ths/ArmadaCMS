import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  DeleteButton,
  EditButton,
  ListProps,
  NumberField,
  TextField,
} from "react-admin";

export const TimelineEraList = (props: ListProps) => (
  <ResourceList {...props} sort={{ field: "sortOrder", order: "ASC" }}>
    <Datagrid>
      <TextField source="title" label="Title" sortable={false} />
      <NumberField source="sortOrder" label="Sort order" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
