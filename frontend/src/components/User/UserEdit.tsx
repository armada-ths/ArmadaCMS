import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
} from "react-admin";

export const UserEdit = (props: EditProps) => {
  return (
    <Edit
      {...props}
      queryOptions={{
        select: (data: Record<string, unknown>) => ({
          ...data,
          role_ids: ((data.roles ?? []) as Array<{ id: number }>).map(
            (r) => r.id,
          ),
        }),
      }}
    >
      <SimpleForm>
        <TextInput label="Username" source="username" />
        <TextInput label="New password" source="password" type="password" />
        <TextInput label="Name" source="name" />
        <TextInput label="Avatar" source="avatar" />
        <ReferenceArrayInput source="role_ids" reference="roles">
          <SelectArrayInput label="Roles" optionText="name" />
        </ReferenceArrayInput>
      </SimpleForm>
    </Edit>
  );
};
