import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
  // FileField,
} from "react-admin";

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
        {/* <FileField source="photo" /> */}
        <EditButton />
        <DeleteWithConfirmButton
          confirmTitle="Are you sure?"
          confirmContent="This is PERMANENT, no backups"
        />
      </Datagrid>
    </List>
  );
};
