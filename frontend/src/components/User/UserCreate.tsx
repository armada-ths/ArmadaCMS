import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
} from "react-admin";

export const UserCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="Password" source="password" type="password" />
        <TextInput label="Name" source="name" />
        <TextInput label="Avatar" source="avatar" />
        <ReferenceArrayInput source="role_ids" reference="roles">
          <SelectArrayInput label="Roles" optionText="name" />
        </ReferenceArrayInput>
      </SimpleForm>
    </Create>
  );
};
