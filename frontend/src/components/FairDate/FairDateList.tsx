import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const FairDateList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="description" label="Description" />
      <TextField source="fairDays" label="Fair Days" />
      <TextField source="irStart" label="IR Start" />
      <TextField source="irEnd" label="IR End" />
      <TextField source="frStart" label="FR Start" />
      <TextField source="frEnd" label="FR End" />
      <EditButton />
      <DeleteWithConfirmButton
        confirmTitle="Are you sure?"
        confirmContent="This is PERMANENT"
      />
    </Datagrid>
  </List>
);
