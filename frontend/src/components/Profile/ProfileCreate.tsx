import {
  Create,
  CreateProps,
  FileInput,
  SelectInput,
  SimpleForm,
  TextInput,
  ReferenceInput,
} from "react-admin";

export const ProfileCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Name" source="name" />
        <TextInput label="Title" source="title" />
        <ReferenceInput label="Team" source="team_id" reference="teams">
          <SelectInput optionText="team_name" />
        </ReferenceInput>
        <TextInput label="Linkedin" source="linkedin" />
        <TextInput label="Email" source="email" type="email" />
        <FileInput label="Photo" source="photo" />
      </SimpleForm>
    </Create>
  );
};
