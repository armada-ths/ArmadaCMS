import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const UserEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="New password" source="password" type="password" />
        <TextInput label="Name" source="name" />
        <TextInput label="Avatar" source="avatar" />
      </SimpleForm>
    </Edit>
  );
};
