import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ReferenceInput,
  SelectInput,
  ImageInput,
  ImageField,
} from "react-admin";
import React, { useState } from "react";

const RANK_CHOICES = [
  { id: "Project Manager", name: "Project Manager" },
  { id: "Project Group", name: "Project Group" },
  { id: "Operation Team", name: "Operation Team" },
  { id: "Host", name: "Host" },
];

export const ProfileEdit = (props: EditProps) => {
  const [selectedOption, setSelectedOption] = useState<
    "upload" | "link" | "display"
  >("display");

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "upload" || event.target.value === "link") {
      setSelectedOption(event.target.value);
    }
  };

  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput label="Name" source="name" />
        <SelectInput
          label="Rank"
          source="rank"
          choices={RANK_CHOICES}
          emptyText="No rank"
          format={(value) => value ?? ""}
          parse={(value) => (value === "" ? null : value)}
        />
        <TextInput label="Title" source="title" />
        <ReferenceInput label="Team" source="team_id" reference="teams">
          <SelectInput
            optionText="team_name"
            emptyText="No team"
            format={(value) => value ?? ""}
            parse={(value) => (value === "" ? null : value)}
          />
        </ReferenceInput>
        <TextInput label="Linkedin" source="linkedin" />
        <TextInput label="Email" source="email" type="email" />

        {/* Radio selection */}
        <div>
          <label>
            <input
              type="radio"
              value="upload"
              checked={selectedOption === "upload"}
              onChange={handleChange}
            />
            Upload new photo
          </label>

          <label style={{ marginLeft: "1em" }}>
            <input
              type="radio"
              value="link"
              checked={selectedOption === "link"}
              onChange={handleChange}
            />
            Enter link to photo
          </label>
        </div>

        {/* Show input based on selected option */}
        {selectedOption === "upload" && (
          <ImageInput label="Photo" source="file">
            <ImageField source="src" title="title" />
          </ImageInput>
        )}
        {selectedOption === "link" && (
          <TextInput label="Photo URL" source="photoUrl" />
        )}
        <div style={{ marginBottom: "1em" }}>
          <p>Current photo:</p>
          <ImageField source="photo" label="Current photo" />
        </div>
      </SimpleForm>
    </Edit>
  );
};
