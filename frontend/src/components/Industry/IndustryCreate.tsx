import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

// test comment to trigger CD

export const IndustryCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
    </SimpleForm>
  </Create>
);
