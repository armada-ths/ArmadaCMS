import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const TimelineEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Title" source="timeline_title" />
        <TextInput type="date" label="Date" source="timeline_date" />
      </SimpleForm>
    </Edit>
  );
};
