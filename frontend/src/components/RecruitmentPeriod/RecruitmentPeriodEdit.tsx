import {
  DateTimeInput,
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
} from "react-admin";

export const RecruitmentPeriodEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" fullWidth />
      <TextInput source="link" fullWidth />
      <DateTimeInput source="startDate" />
      <DateTimeInput source="endDate" />
    </SimpleForm>
  </Edit>
);
