import {
  NumberInput,
  ReferenceInput,
  SelectInput,
  SimpleForm,
  TextInput,
  required,
} from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";

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
    <MarkdownInput
      source="body"
      label="Body (Markdown)"
      validate={required()}
      enableImageFeatures={false}
    />
    <NumberInput
      source="sortOrder"
      label="Sort order"
      helperText="Order within the era. Equal numbers are shown in entry ID order."
      validate={required()}
    />
  </SimpleForm>
);
