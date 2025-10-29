import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const ProgramEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" />
    </SimpleForm>
  </Edit>
);
