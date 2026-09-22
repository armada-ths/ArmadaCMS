import {
  NumberInput,
  ReferenceInput,
  SelectInput,
  SimpleForm,
  TextInput,
  required,
} from "react-admin";

export const TimelineEntryForm = () => (
  <SimpleForm>
    <ReferenceInput
      source="eraId"
      reference="timeline-eras"
      sort={{ field: "sortOrder", order: "ASC" }}
    >
      <SelectInput
        optionText="title"
        label="Era"
        helperText="Choose the timeline section for this entry."
        validate={required()}
      />
    </ReferenceInput>
    <TextInput
      source="title"
      label="Title"
      helperText="Heading displayed on the timeline."
      validate={required()}
      fullWidth
    />
    <TextInput
      source="body"
      label="Body"
      helperText="Description displayed under the heading."
      validate={required()}
      multiline
      rows={4}
      fullWidth
    />
    <NumberInput
      source="sortOrder"
      label="Sort order"
      helperText="Order within the era. Equal numbers are shown in entry ID order."
      validate={required()}
    />
  </SimpleForm>
);
