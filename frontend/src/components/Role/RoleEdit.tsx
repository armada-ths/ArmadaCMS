import {
  ArrayInput,
  Edit,
  EditProps,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  ChangeOwnPasswordInput,
  PermissionActionsInput,
} from "./RolePermissionInputs";
import { normalizeRecord, transformRole } from "./rolePermissionUtils";

export const RoleEdit = (props: EditProps) => (
  <Edit
    {...props}
    queryOptions={{ select: normalizeRecord }}
    transform={transformRole}
  >
    <SimpleForm>
      <TextInput label="Role name" source="name" fullWidth />

      <ArrayInput source="permissions" label="Permissions">
        <SimpleFormIterator disableReordering>
          <PermissionActionsInput />
        </SimpleFormIterator>
      </ArrayInput>

      <ChangeOwnPasswordInput />
    </SimpleForm>
  </Edit>
);
