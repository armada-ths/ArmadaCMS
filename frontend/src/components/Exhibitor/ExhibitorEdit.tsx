import { useState } from "react";
import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
  BooleanInput,
  ImageInput,
  ImageField,
} from "react-admin";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

export interface Program {
  id: number;
  code: string;
  name: string;
}
export interface Industry {
  id: number;
  name: string;
}
export interface Employment {
  id: number;
  name: string;
}
export const ExhibitorEdit = (props: EditProps) => {
  const [selectedOption, setSelectedOption] = useState<"upload" | "link">(
    "upload",
  );

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "upload" || event.target.value === "link") {
      setSelectedOption(event.target.value);
    }
  };

  return (
    <Edit
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
        employments: (data.employments || [])
          .map((e: Employment) =>
            typeof e === "number" ? { id: e } : e?.id ? { id: e.id } : null,
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
        <div>
          <label>
            <input
              type="radio"
              value="upload"
              checked={selectedOption === "upload"}
              onChange={handleChange}
            />
            Upload photo
          </label>

          <label>
            <input
              type="radio"
              value="link"
              checked={selectedOption === "link"}
              onChange={handleChange}
            />
            Enter link to existing photo
          </label>
        </div>
        {selectedOption === "upload" ? (
          <ImageInput
            label="Logo freesize"
            source="logoFreesize"
            accept={IMAGE_INPUT_ACCEPT}
            validate={validateImageUpload}
          >
            <ImageField source="src" title="title" />
          </ImageInput>
        ) : (
          <TextInput label="Logo freesize" source="logoFreesize" />
        )}
        <div style={{ marginBottom: "1em" }}>
          <p>Current image:</p>
          <ImageField source="logoFreesize" label="Current Logo" />
        </div>
        <TextInput source="mapImg" />

        <ReferenceArrayInput source="industries" reference="industries">
          <SelectArrayInput optionText="name" />
        </ReferenceArrayInput>

        <ReferenceArrayInput source="programs" reference="programs">
          <SelectArrayInput optionText="name" />
        </ReferenceArrayInput>

        <ReferenceArrayInput source="employments" reference="employments">
          <SelectArrayInput optionText="name" />
        </ReferenceArrayInput>

        <TextInput source="cities" />
        <TextInput source="fairLocation" />
        <TextInput source="vyerPosition" />
        <TextInput source="locationSpecial" />
        <BooleanInput source="climateCompensation" />
        <TextInput source="flyer" />
      </SimpleForm>
    </Edit>
  );
};
