import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const TeamList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="team_name" />
        <EditButton />
        <DeleteWithConfirmButton
          confirmTitle="Are you sure?"
          confirmContent="This is PERMANENT, no backups"
        />
      </Datagrid>
    </List>
  );
};
