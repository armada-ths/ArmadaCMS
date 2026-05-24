import {
  ArrayInput,
  Edit,
  EditProps,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  PermissionActionsInput,
  SpecialPermissionInput,
} from "./RolePermissionInputs";
import {
  normalizeRecord,
  SPECIAL_PERMISSIONS,
  transformRole,
} from "./rolePermissionUtils";

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

      {SPECIAL_PERMISSIONS.map((spec) => (
        <SpecialPermissionInput key={spec.formField} spec={spec} />
      ))}
    </SimpleForm>
  </Edit>
);
