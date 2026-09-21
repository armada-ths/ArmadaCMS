import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const IndustryCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
    </SimpleForm>
  </Create>
);
