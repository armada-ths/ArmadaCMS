import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const TeamList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="team_name" />
        <EditButton />
        <DeleteButton />
      </Datagrid>
    </List>
  );
};
