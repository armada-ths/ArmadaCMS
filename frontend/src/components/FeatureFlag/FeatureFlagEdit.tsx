import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  BooleanInput,
  Labeled,
} from "react-admin";
import { AutoOverrideBadge } from "./AutoOverrideBadge";

export const FeatureFlagEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="key" disabled />
      <TextInput source="description" />
      <BooleanInput source="enabled" />
      <Labeled label="Auto override">
        <AutoOverrideBadge />
      </Labeled>
    </SimpleForm>
  </Edit>
);
