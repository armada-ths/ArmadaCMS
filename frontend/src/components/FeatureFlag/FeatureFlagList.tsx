import {
  List,
  Datagrid,
  TextField,
  BooleanField,
  EditButton,
  ListProps,
} from "react-admin";
import { AutoOverrideBadge } from "./AutoOverrideBadge";

export const FeatureFlagList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="key" />
      <TextField source="description" />
      <BooleanField source="enabled" />
      <AutoOverrideBadge />
      <EditButton />
    </Datagrid>
  </List>
);
