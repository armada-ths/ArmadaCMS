import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ArrayInput,
  SimpleFormIterator,
} from "react-admin";

export const RoleEdit = (props: EditProps) => (
  <Edit {...props}>
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
  </Edit>
);
