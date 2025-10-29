import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const EmploymentCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
    </SimpleForm>
  </Create>
);
