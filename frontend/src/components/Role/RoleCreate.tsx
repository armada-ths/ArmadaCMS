import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ArrayInput,
  SimpleFormIterator,
  SelectInput,
  FormDataConsumer,
} from "react-admin";

const resources = [
  { id: "*", name: "Full permission (*)" },
  { id: "profiles", name: "Profiles" },
  { id: "teams", name: "Teams" },
  { id: "exhibitors", name: "Exhibitors" },
  { id: "events", name: "Events" },
  { id: "dates", name: "Dates" },
  { id: "roles", name: "Roles" },
];

const actions = [
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

const transformRole = (data: any) => ({
  ...data,
  permissions: (data.permissions ?? [])
    .map((p: any) => {
      if (typeof p === "string") return p;
      if (p?.resource === "*" || p?.action === "*") return "*";
      if (p?.resource && p?.action) return `${p.resource}.${p.action}`;
      return null;
    })
    .filter(Boolean),
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
            {({ scopedFormData }) =>
              scopedFormData?.resource && scopedFormData?.action ? (
                <span style={{ marginTop: 28 }}>
                  {scopedFormData.resource === "*" ||
                  scopedFormData.action === "*"
                    ? "*"
                    : `${scopedFormData.resource}.${scopedFormData.action}`}
                </span>
              ) : null
            }
          </FormDataConsumer>
        </SimpleFormIterator>
      </ArrayInput>
    </SimpleForm>
  </Create>
);
