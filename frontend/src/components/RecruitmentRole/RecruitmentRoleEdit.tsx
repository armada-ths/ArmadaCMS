import {
  Edit,
  EditProps,
  ReferenceInput,
  SelectInput,
  SimpleForm,
  TextInput,
} from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";

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
      <MarkdownInput
        source="description"
        label="Description (Markdown)"
        enableImageFeatures={false}
      />
    </SimpleForm>
  </Edit>
);
