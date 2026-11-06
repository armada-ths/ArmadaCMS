import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const TeamEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Team name" source="team_name" />
        <TextInput label="Team name (Swedish)" source="team_name_sv" />
      </SimpleForm>
    </Edit>
  );
};
