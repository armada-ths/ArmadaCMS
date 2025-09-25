import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  DateTimeInput,
  BooleanInput,
  NumberInput,
} from "react-admin";

export const EventCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="location" />
      <TextInput source="description" />
      <TextInput source="food" />
      <DateTimeInput source="eventStart" />
      <DateTimeInput source="eventEnd" />
      <DateTimeInput source="registrationEnd" />
      <TextInput source="fee" />
      <BooleanInput source="registrationRequired" />
      <TextInput source="signupLink" />
      <NumberInput source="eventMaxCapacity" />
    </SimpleForm>
  </Create>
);
