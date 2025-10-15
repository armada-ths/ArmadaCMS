import { useState } from "react";
import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  DateTimeInput,
  BooleanInput,
  NumberInput,
  ImageInput,
  ImageField,
} from "react-admin";
import { toLocalInputValue, toUTCISOString } from "@/utils/dateTimeHelpers";

export const EventCreate = (props: CreateProps) => {
  const [selectedOption, setSelectedOption] = useState<"upload" | "link">(
    "upload",
  );

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "upload" || event.target.value === "link") {
      setSelectedOption(event.target.value);
    }
  };

  return (
    <Create {...props}>
      <SimpleForm>
        <TextInput source="name" />
        <TextInput source="location" />
        <TextInput source="description" />
        <TextInput source="food" />
        <DateTimeInput
          source="eventStart"
          label="Event Start (Stockholm time)"
          parse={toUTCISOString}
          format={toLocalInputValue}
        />

        <DateTimeInput
          source="eventEnd"
          label="Event End (Stockholm time)"
          parse={toUTCISOString}
          format={toLocalInputValue}
        />

        <DateTimeInput
          source="registrationEnd"
          label="Registration End (Stockholm time)"
          parse={toUTCISOString}
          format={toLocalInputValue}
        />
        <TextInput source="fee" />
        <BooleanInput source="registrationRequired" />
        <TextInput source="signupLink" />
        <NumberInput source="eventMaxCapacity" />
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
          <ImageInput label="Photo" source="imageFile">
            <ImageField source="src" title="title" />
          </ImageInput>
        ) : (
          <TextInput label="URL" source="imageUrl" />
        )}
      </SimpleForm>
    </Create>
  );
};
