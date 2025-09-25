import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
  BooleanInput,
} from "react-admin";

export const ExhibitorCreate = (props: CreateProps) => (
  <Create {...props}>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="type" />
      <TextInput source="tier" />
      <TextInput source="companyWebsite" />
      <TextInput source="about" />
      <TextInput source="purpose" />
      <TextInput source="logoSquared" />
      <TextInput source="logoFreesize" />
      <TextInput source="mapImg" />
      <ReferenceArrayInput source="industries" reference="industries">
        <SelectArrayInput optionText="name" />
      </ReferenceArrayInput>
      <ReferenceArrayInput source="programs" reference="programs">
        <SelectArrayInput optionText="name" />
      </ReferenceArrayInput>
      <TextInput source="cities" />
      <TextInput source="fairLocation" />
      <TextInput source="vyerPosition" />
      <TextInput source="locationSpecial" />
      <BooleanInput source="climateCompensation" />
      <TextInput source="flyer" />
    </SimpleForm>
  </Create>
);
