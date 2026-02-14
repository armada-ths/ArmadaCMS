import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  ArrayField,
  SingleFieldList,
  ChipField,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const RoleList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <ArrayField source="permissions">
        <SingleFieldList linkType={false}>
          <ChipField source="" />
        </SingleFieldList>
      </ArrayField>
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
