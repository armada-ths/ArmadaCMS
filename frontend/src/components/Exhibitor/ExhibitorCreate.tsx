import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
  BooleanInput,
} from "react-admin";
import { Industry, Program } from "./ExhibitorEdit";

export const ExhibitorCreate = (props: CreateProps) => (
  // <Create {...props}>
  <Create
    {...props}
    transform={(data) => ({
      ...data,
      programs: (data.programs || [])
        .map((p: Program) =>
          typeof p === "number" ? { id: p } : p?.id ? { id: p.id } : null,
        )
        .filter(Boolean),
      industries: (data.industries || [])
        .map((i: Industry) =>
          typeof i === "number" ? { id: i } : i?.id ? { id: i.id } : null,
        )
        .filter(Boolean),
    })}
  >
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
