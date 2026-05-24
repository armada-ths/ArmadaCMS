import {
  ArrayInput,
  Create,
  CreateProps,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  PermissionActionsInput,
  SpecialPermissionInput,
} from "./RolePermissionInputs";
import { SPECIAL_PERMISSIONS, transformRole } from "./rolePermissionUtils";

export const RoleCreate = (props: CreateProps) => (
  <Create {...props} transform={transformRole}>
    <SimpleForm>
      <TextInput label="Role name" source="name" fullWidth />

      <ArrayInput source="permissions" label="Permissions">
        <SimpleFormIterator disableReordering>
          <PermissionActionsInput />
        </SimpleFormIterator>
      </ArrayInput>

      {SPECIAL_PERMISSIONS.map((spec) => (
        <SpecialPermissionInput key={spec.formField} spec={spec} />
      ))}
    </SimpleForm>
  </Create>
);
