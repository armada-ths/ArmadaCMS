import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  BooleanInput,
} from "react-admin";

export const FeatureFlagEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="key" disabled />
      <TextInput source="description" />
      <BooleanInput source="enabled" />
    </SimpleForm>
  </Edit>
);
