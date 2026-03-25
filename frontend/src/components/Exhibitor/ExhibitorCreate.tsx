import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ReferenceArrayInput,
  SelectArrayInput,
  BooleanInput,
  ImageInput,
  ImageField,
} from "react-admin";
import { Employment, Industry, Program } from "./ExhibitorEdit";
import { useState } from "react";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";
import {
  formatExternalUrlInput,
  normalizeExternalUrl,
  validateExternalUrl,
} from "@/utils/externalLinkGuards";
import { InputAdornment } from "@mui/material";

export const ExhibitorCreate = (props: CreateProps) => {
  const [selectedOption, setSelectedOption] = useState<"upload" | "link">(
    "upload",
  );

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "upload" || event.target.value === "link") {
      setSelectedOption(event.target.value);
    }
  };

  return (
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
        <TextInput
          source="companyWebsite"
          format={formatExternalUrlInput}
          parse={normalizeExternalUrl}
          validate={validateExternalUrl}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">https://</InputAdornment>
              ),
            },
          }}
        />
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
    </Create>
  );
};
