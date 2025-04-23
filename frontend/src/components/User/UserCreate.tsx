import React from "react";
import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const UserCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="Password" source="password" type="password" />
        <TextInput label="Name" source="name" />
        <TextInput label="Avatar" source="avatar" />
      </SimpleForm>
    </Create>
  );
};
