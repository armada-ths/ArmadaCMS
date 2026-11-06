import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const TeamList = (props: ListProps) => {
  return (
    <ResourceList {...props}>
      <Datagrid>
        <TextField source="team_name" />
        <EditButton />
        <DeleteButton />
      </Datagrid>
    </ResourceList>
  );
};
