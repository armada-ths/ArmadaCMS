import {
  ArrayInput,
  Edit,
  EditProps,
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

type RoleRecord = {
  id?: number;
  name?: string;
  permissions?: Array<string | PermissionInput>;
};

const actions = [
  { id: "*", name: "All actions (*)" },
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

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

const parsePermission = (permission: string): PermissionInput => {
  if (permission === "*") {
    return { resource: "*", action: "*" };
  }

  const [resource, action] = permission.split(".");

  return { resource, action };
};

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

const normalizeRecord = (record: RoleRecord): RoleRecord => ({
  ...record,
  permissions: (record.permissions ?? []).map((permission) =>
    typeof permission === "string" ? parsePermission(permission) : permission,
  ),
});

const transformRole = (data: RoleRecord) => ({
  ...data,
  permissions: (data.permissions ?? [])
    .map(toPermission)
    .filter((permission): permission is string => Boolean(permission)),
});

export const RoleEdit = (props: EditProps) => {
  const resources = useResourceChoices();

  return (
    <Edit
      {...props}
      queryOptions={{
        select: normalizeRecord,
      }}
      transform={transformRole}
    >
      <SimpleForm>
        <TextInput label="Name" source="name" fullWidth />

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
    </Edit>
  );
};
