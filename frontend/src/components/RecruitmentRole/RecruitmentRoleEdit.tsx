import {
  Edit,
  EditProps,
  ReferenceInput,
  SelectInput,
  SimpleForm,
  TextInput,
} from "react-admin";

export const RecruitmentRoleEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" fullWidth />
      <ReferenceInput source="team_id" reference="teams">
        <SelectInput optionText="team_name" emptyText="No team" />
      </ReferenceInput>
      <ReferenceInput source="recruitmentId" reference="recruitmentperiods">
        <SelectInput optionText="name" />
      </ReferenceInput>
      <TextInput
        source="description"
        multiline
        fullWidth
        minRows={5}
        helperText="Supports Markdown formatting"
      />
    </SimpleForm>
  </Edit>
);
