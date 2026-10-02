import { NumberInput, SimpleForm, TextInput, required } from "react-admin";

export const TimelineEraForm = () => (
  <SimpleForm>
    <TextInput
      source="title"
      label="Title"
      helperText="Section heading displayed on the timeline."
      validate={required()}
      fullWidth
    />
    <NumberInput
      source="sortOrder"
      label="Sort order"
      helperText="Order of sections on the timeline. Equal numbers are shown in era ID order."
      validate={required()}
    />
  </SimpleForm>
);
