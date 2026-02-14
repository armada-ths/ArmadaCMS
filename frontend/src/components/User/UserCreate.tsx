import React from "react";
import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ReferenceInput,
  SelectInput,
} from "react-admin";

export const UserCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="Password" source="password" type="password" />
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
    </Create>
  );
};
