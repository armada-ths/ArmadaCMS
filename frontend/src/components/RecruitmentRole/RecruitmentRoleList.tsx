import {
  Datagrid,
  DeleteButton,
  EditButton,
  List,
  ListProps,
  ReferenceField,
  TextField,
} from "react-admin";

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
      <DeleteButton />
    </Datagrid>
  </List>
);
