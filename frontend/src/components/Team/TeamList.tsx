import { List, Datagrid, TextField, EditButton, ListProps } from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const TeamList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="team_name" />
        <EditButton />
        <PermissionDeleteButton />
      </Datagrid>
    </List>
  );
};
