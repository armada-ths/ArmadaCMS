import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const ProgramCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
    </SimpleForm>
  </Create>
);
