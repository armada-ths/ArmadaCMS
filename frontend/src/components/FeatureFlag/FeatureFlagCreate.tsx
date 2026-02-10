import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  BooleanInput,
} from "react-admin";

export const FeatureFlagCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="key" />
      <TextInput source="description" />
      <BooleanInput source="enabled" defaultValue={false} />
    </SimpleForm>
  </Create>
);
