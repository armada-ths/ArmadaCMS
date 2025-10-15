import { useState } from "react";
import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  DateTimeInput,
  BooleanInput,
  NumberInput,
  ImageField,
  ImageInput,
} from "react-admin";

export const EventEdit = (props: EditProps) => {
  const [selectedOption, setSelectedOption] = useState<"upload" | "link">(
    "upload",
  );

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "upload" || event.target.value === "link") {
      setSelectedOption(event.target.value);
    }
  };
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput source="name" />
        <TextInput source="location" />
        <TextInput source="description" />
        <TextInput source="food" />
        <DateTimeInput source="eventStart" />
        <DateTimeInput source="eventEnd" />
        <DateTimeInput source="registrationEnd" />
        <NumberInput source="fee" />
        <BooleanInput source="registrationRequired" />
        <TextInput source="signupLink" />
        <NumberInput source="eventMaxCapacity" />
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
          <TextInput label="Photo URL" source="imageUrl" />
        )}
        <div style={{ marginBottom: "1em" }}>
          <p>Current image:</p>
          <ImageField source="imageUrl" label="Current photo" />
        </div>
      </SimpleForm>
    </Edit>
  );
};
