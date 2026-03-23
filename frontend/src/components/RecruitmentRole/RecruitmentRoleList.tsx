import {
  Datagrid,
  EditButton,
  List,
  ListProps,
  ReferenceField,
  TextField,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const RecruitmentRoleList = (props: ListProps) => (
  <List {...props}>
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
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
