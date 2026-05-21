import {
  ArrayInput,
  Create,
  CreateProps,
  FormDataConsumer,
  SelectInput,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
  useResourceDefinitions,
} from "react-admin";

type PermissionInput = {
  resource?: string;
  action?: string;
};

type RoleFormData = {
  name?: string;
  permissions?: PermissionInput[];
};

const actions = [
  { id: "*", name: "All actions (*)" },
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

const toPermission = (permission: string | PermissionInput): string | null => {
  if (typeof permission === "string") {
    return permission;
  }

  if (!permission.resource || !permission.action) {
    return null;
  }

  if (permission.resource === "*" && permission.action === "*") {
    return "*";
  }

  return `${permission.resource}.${permission.action}`;
};
const transformRole = (data: RoleFormData) => ({
  ...data,
  permissions: (data.permissions ?? [])
    .map(toPermission)
    .filter((permission): permission is string => Boolean(permission)),
});

const useResourceChoices = () => {
  const resourceDefinitions = useResourceDefinitions();

  return [
    { id: "*", name: "All resources (*)" },
    ...Object.entries(resourceDefinitions).map(([resource, definition]) => ({
      id: resource,
      name: definition.options?.label ?? resource,
    })),
  ];
};

export const RoleCreate = (props: CreateProps) => {
  const resources = useResourceChoices();

  return (
    <Create {...props} transform={transformRole}>
      <SimpleForm>
        <TextInput label="Role name" source="name" fullWidth />

        <ArrayInput source="permissions" label="Permissions">
          <SimpleFormIterator inline>
            <SelectInput
              source="resource"
              label="Resource"
              choices={resources}
            />
            <SelectInput source="action" label="Action" choices={actions} />

            <FormDataConsumer>
              {({ scopedFormData }) => {
                const permission = scopedFormData
                  ? toPermission(scopedFormData as PermissionInput)
                  : null;

                return permission ? <span>{permission}</span> : null;
              }}
            </FormDataConsumer>
          </SimpleFormIterator>
        </ArrayInput>
      </SimpleForm>
    </Create>
  );
};
