import {
  Create,
  CreateProps,
  SelectInput,
  SimpleForm,
  TextInput,
  ReferenceInput,
  ImageInput,
  ImageField,
} from "react-admin";
import React, { useState } from "react";

export const ProfileCreate = (props: CreateProps) => {
  const [selectedOption, setSelectedOption] = useState<"upload" | "link">(
    "upload",
  );

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value == "upload" || event.target.value == "link")
      setSelectedOption(event.target.value);
  };

  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput label="Name" source="name" />
        <TextInput label="Title" source="title" />
        <ReferenceInput label="Team" source="team_id" reference="teams">
          <SelectInput optionText="team_name" />
        </ReferenceInput>
        <TextInput label="Linkedin" source="linkedin" />
        <TextInput label="Email" source="email" type="email" />
        {/* <FileInput label="Photo" source="photo" /> */}
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
          <ImageInput label="Photo" source="photoFile">
            <ImageField source="src" title="title" />
          </ImageInput>
        ) : (
          <TextInput label="URL" source="photoUrl" />
        )}
      </SimpleForm>
    </Create>
  );
};
