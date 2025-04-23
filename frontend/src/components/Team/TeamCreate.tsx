import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const TeamCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Team name" source="team_name" />
      </SimpleForm>
    </Create>
  );
};
