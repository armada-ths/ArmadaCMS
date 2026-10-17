import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const TeamCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Team name" source="team_name" />
        <TextInput label="Team name (Swedish)" source="team_name_sv" />
      </SimpleForm>
    </Create>
  );
};
