import {
  ArrayInput,
  Create,
  CreateProps,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  ChangeOwnPasswordInput,
  PermissionActionsInput,
} from "./RolePermissionInputs";
import { transformRole } from "./rolePermissionUtils";

export const RoleCreate = (props: CreateProps) => (
  <Create {...props} transform={transformRole}>
    <SimpleForm>
      <TextInput label="Role name" source="name" fullWidth />

      <ArrayInput source="permissions" label="Permissions">
        <SimpleFormIterator>
          <PermissionActionsInput />
        </SimpleFormIterator>
      </ArrayInput>

      <ChangeOwnPasswordInput />
    </SimpleForm>
  </Create>
);
