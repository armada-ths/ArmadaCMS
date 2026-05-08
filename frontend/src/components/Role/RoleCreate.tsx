import {
  ArrayInput,
  Create,
  CreateProps,
  FormDataConsumer,
  SelectInput,
  SimpleForm,
  SimpleFormIterator,
  TextInput,
} from "react-admin";

type PermissionInput = {
  resource?: string;
  action?: string;
};

type RoleFormData = {
  name?: string;
  permissions?: PermissionInput[];
};

const resources = [
  { id: "*", name: "All resources (*)" },
  { id: "profiles", name: "Profiles" },
  { id: "teams", name: "Teams" },
  { id: "exhibitors", name: "Exhibitors" },
  { id: "events", name: "Events" },
  { id: "dates", name: "Dates" },
  { id: "roles", name: "Roles" },
  { id: "customusers", name: "Custom users" },
];

const actions = [
  { id: "*", name: "All actions (*)" },
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

const toPermission = (permission: PermissionInput): string | null => {
  if (permission.resource === "*" || permission.action === "*") {
    return "*";
  }

  if (permission.resource && permission.action) {
    return `${permission.resource}.${permission.action}`;
  }

  return null;
};

const transformRole = (data: RoleFormData) => ({
  ...data,
  permissions: (data.permissions ?? [])
    .map(toPermission)
    .filter((permission): permission is string => Boolean(permission)),
});

export const RoleCreate = (props: CreateProps) => (
  <Create {...props} transform={transformRole}>
    <SimpleForm>
      <TextInput label="Role name" source="name" fullWidth />

      <ArrayInput source="permissions" label="Permissions">
        <SimpleFormIterator inline>
          <SelectInput source="resource" label="Resource" choices={resources} />
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
