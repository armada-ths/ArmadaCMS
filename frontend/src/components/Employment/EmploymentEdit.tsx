import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const EmploymentEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="nameSv" label="Name (Swedish)" />
    </SimpleForm>
  </Edit>
);
