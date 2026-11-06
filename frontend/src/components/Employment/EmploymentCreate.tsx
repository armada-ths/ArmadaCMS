import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const EmploymentCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="nameSv" label="Name (Swedish)" />
    </SimpleForm>
  </Create>
);
