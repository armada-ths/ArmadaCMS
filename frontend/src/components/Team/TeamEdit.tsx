import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const TeamEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Team name" source="team_name" />
      </SimpleForm>
    </Edit>
  );
};
