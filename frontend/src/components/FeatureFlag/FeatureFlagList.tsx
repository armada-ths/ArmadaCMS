import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  BooleanField,
  EditButton,
  ListProps,
} from "react-admin";

export const FeatureFlagList = (props: ListProps) => (
  <ResourceList {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="key" />
      <TextField source="description" />
      <BooleanField source="enabled" />
      <EditButton />
    </Datagrid>
  </ResourceList>
);
