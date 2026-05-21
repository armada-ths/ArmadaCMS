import {
  ArrayInput,
  BooleanInput,
  CheckboxGroupInput,
  Edit,
  EditProps,
  SelectInput,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  PERMISSION_ACTIONS,
  normalizeRecord,
  transformRole,
  useResourceChoices,
} from "./rolePermissionUtils";

export const RoleEdit = (props: EditProps) => {
  const resources = useResourceChoices();

  return (
    <Edit
      {...props}
      queryOptions={{ select: normalizeRecord }}
      transform={transformRole}
    >
      <SimpleForm>
        <TextInput label="Role name" source="name" fullWidth />

        <ArrayInput source="permissions" label="Permissions">
          <SimpleFormIterator>
            <SelectInput
              source="resource"
              label="Resource"
              choices={resources}
            />
            <CheckboxGroupInput
              source="actions"
              label="Actions"
              choices={PERMISSION_ACTIONS}
            />
          </SimpleFormIterator>
        </ArrayInput>

        <BooleanInput
          source="changeOwnPassword"
          label="Can change own password (customusers.changeownpassword)"
        />
      </SimpleForm>
    </Edit>
  );
};
