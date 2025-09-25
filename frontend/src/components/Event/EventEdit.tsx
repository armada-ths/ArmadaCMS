import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  DateTimeInput,
  BooleanInput,
  NumberInput,
} from "react-admin";

export const EventEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="location" />
      <TextInput source="description" />
      <TextInput source="food" />
      <DateTimeInput source="eventStart" />
      <DateTimeInput source="eventEnd" />
      <DateTimeInput source="registrationEnd" />
      <NumberInput source="fee" />
      <BooleanInput source="registrationRequired" />
      <TextInput source="signupLink" />
      <NumberInput source="eventMaxCapacity" />
    </SimpleForm>
  </Edit>
);
