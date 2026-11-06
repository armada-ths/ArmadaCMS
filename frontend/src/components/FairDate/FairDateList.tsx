import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const FairDateList = (props: ListProps) => (
  <ResourceList {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="description" label="Description" />
      <TextField source="fairDays" label="Fair Days" />
      <TextField source="irStart" label="IR Start" />
      <TextField source="irEnd" label="IR End" />
      <TextField source="frStart" label="FR Start" />
      <TextField source="frEnd" label="FR End" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
