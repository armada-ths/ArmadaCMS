import { List, Datagrid, TextField, EditButton, ListProps } from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const ProfileList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="name" />
        <TextField source="rank" />
        <TextField source="title" />
        <TextField source="team.team_name" label="Team name" />
        <TextField source="linkedin" />
        <TextField source="email" />
        <EditButton />
        <PermissionDeleteButton />
      </Datagrid>
    </List>
  );
};
