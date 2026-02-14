import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ReferenceInput,
  SelectInput,
} from "react-admin";

export const UserEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="New password" source="password" type="password" />
        <TextInput label="Name" source="name" />
        <TextInput label="Avatar" source="avatar" />
        <ReferenceInput source="role_id" reference="roles">
          <SelectInput
            label="Role"
            optionText="name"
            helperText="Leave empty for full admin access"
          />
        </ReferenceInput>
      </SimpleForm>
    </Edit>
  );
};
