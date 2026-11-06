import {
  ReferenceInput,
  SelectInput,
  NumberInput,
  SimpleForm,
  TextInput,
  minValue,
  required,
} from "react-admin";

const integer = (value: number | null | undefined) =>
  value == null || Number.isSafeInteger(value)
    ? undefined
    : "Enter a whole number";

export const LoungeItemForm = () => (
  <SimpleForm defaultValues={{ soldOut: false }}>
    <ReferenceInput
      source="categoryId"
      reference="lounge-categories"
      perPage={1000}
    >
      <SelectInput optionText="name" label="Category" validate={required()} />
    </ReferenceInput>
    <TextInput
      source="name"
      label="Name (English)"
      validate={required()}
      fullWidth
    />
    <TextInput source="nameSv" label="Name (Swedish)" fullWidth />
    <TextInput
      source="description"
      label="Description (English)"
      multiline
      fullWidth
    />
    <TextInput
      source="descriptionSv"
      label="Description (Swedish)"
      multiline
      fullWidth
    />
    <NumberInput
      source="maxPerOrder"
      label="Maximum per order"
      helperText="Optional. Leave blank for no per-item limit."
      validate={[minValue(1), integer]}
    />
  </SimpleForm>
);
