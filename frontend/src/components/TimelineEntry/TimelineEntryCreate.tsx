import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  NumberInput,
} from "react-admin";

export const TimelineEntryCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput source="title" />
        <TextInput source="body" multiline />
        <TextInput source="era" label="Era (e.g. 1980s)" />
        <TextInput source="eraTitle" label="Era title" />
        <NumberInput source="sortOrder" label="Sort order" />
      </SimpleForm>
    </Create>
  );
};
