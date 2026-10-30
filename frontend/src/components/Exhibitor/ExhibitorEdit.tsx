import { useMemo, useState, type ReactNode } from "react";
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
  useRecordContext,
  type Identifier,
} from "react-admin";
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
import { ExhibitorTierInput } from "./ExhibitorTierInput";

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

// The API embeds named records for list views; reference inputs require IDs.
const referenceIds = (values: unknown): Identifier[] =>
  Array.isArray(values)
    ? values.flatMap((value) => {
        const id =
          typeof value === "object" && value !== null ? value.id : value;
        return typeof id === "number" || typeof id === "string" ? [id] : [];
      })
    : [];

const ExhibitorEditForm = ({ children }: { children: ReactNode }) => {
  const record = useRecordContext();
  const formRecord = useMemo(
    () =>
      record
        ? {
            ...record,
            industries: referenceIds(record.industries),
            programs: referenceIds(record.programs),
            employments: referenceIds(record.employments),
          }
        : undefined,
    [record],
  );

  return <SimpleForm record={formRecord}>{children}</SimpleForm>;
};
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
      <ExhibitorEditForm>
        <TextInput source="name" />
        <TextInput source="type" />
        <ExhibitorTierInput />
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
      </ExhibitorEditForm>
    </Edit>
  );
};
