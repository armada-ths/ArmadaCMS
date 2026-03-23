import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  ReferenceField,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const ProfileList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="name" />
        <TextField source="rank" />
        <TextField source="title" />
        <ReferenceField source="team_id" reference="teams" label="Team name">
          <TextField source="team_name" />
        </ReferenceField>
        <TextField source="linkedin" />
        <TextField source="email" />
        <EditButton />
        <PermissionDeleteButton />
      </Datagrid>
    </List>
  );
};
