import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  DeleteButton,
  EditButton,
  ListProps,
  ReferenceField,
  TextField,
} from "react-admin";

export const RecruitmentRoleList = (props: ListProps) => (
  <ResourceList {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <ReferenceField source="team_id" reference="teams" label="Team">
        <TextField source="team_name" />
      </ReferenceField>
      <ReferenceField
        source="recruitmentId"
        reference="recruitmentperiods"
        label="Recruitment period"
      >
        <TextField source="name" />
      </ReferenceField>
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
