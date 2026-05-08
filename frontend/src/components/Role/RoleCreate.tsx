import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ArrayInput,
  SimpleFormIterator,
} from "react-admin";

export const RoleCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput label="Name" source="name" />
      <ArrayInput source="permissions" label="Permissions">
        <SimpleFormIterator inline>
          <TextInput
            source=""
            label="Permission"
            helperText="e.g. profiles.edit"
          />
        </SimpleFormIterator>
      </ArrayInput>
    </SimpleForm>
  </Create>
);
