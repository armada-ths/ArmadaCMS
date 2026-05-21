import {
  ArrayInput,
  BooleanInput,
  CheckboxGroupInput,
  Create,
  CreateProps,
  SelectInput,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";
import {
  PERMISSION_ACTIONS,
  transformRole,
  useResourceChoices,
} from "./rolePermissionUtils";

export const RoleCreate = (props: CreateProps) => {
  const resources = useResourceChoices();

  return (
    <Create {...props} transform={transformRole}>
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
    </Create>
  );
};
