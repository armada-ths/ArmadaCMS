import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const IndustryEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="nameSv" label="Name (Swedish)" />
    </SimpleForm>
  </Edit>
);
