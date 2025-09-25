import { Create, CreateProps, SimpleForm, TextInput } from "react-admin";

export const StartEndCreate = (props: CreateProps) => {
  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Title" source="timeline_title" />
        <TextInput type="date" label="Date" source="timeline_date" />
      </SimpleForm>
    </Create>
  );
};
