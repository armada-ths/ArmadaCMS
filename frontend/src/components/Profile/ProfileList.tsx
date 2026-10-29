import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
  ReferenceField,
} from "react-admin";

export const ProfileList = (props: ListProps) => {
  return (
    <ResourceList {...props}>
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
        <DeleteButton />
      </Datagrid>
    </ResourceList>
  );
};
