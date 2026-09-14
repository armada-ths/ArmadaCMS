import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  NumberInput,
} from "react-admin";

export const TimelineEntryEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput source="title" />
        <TextInput source="body" multiline />
        <TextInput source="era" label="Era (e.g. 1980s)" />
        <TextInput source="eraTitle" label="Era title" />
        <NumberInput source="sortOrder" label="Sort order" />
      </SimpleForm>
    </Edit>
  );
};
