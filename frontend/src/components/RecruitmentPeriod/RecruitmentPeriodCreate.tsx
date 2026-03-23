import {
  Create,
  CreateProps,
  DateTimeInput,
  SimpleForm,
  TextInput,
} from "react-admin";

export const RecruitmentPeriodCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" fullWidth />
      <TextInput source="link" fullWidth />
      <DateTimeInput source="startDate" />
      <DateTimeInput source="endDate" />
    </SimpleForm>
  </Create>
);
