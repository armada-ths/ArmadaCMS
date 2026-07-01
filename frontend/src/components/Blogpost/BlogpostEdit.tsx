import { useState } from "react";
import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  ImageInput,
  ImageField,
  BooleanInput,
} from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

export const BlogpostEdit = (props: EditProps) => {
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
        <TextInput source="title" fullWidth />
        <TextInput source="author" fullWidth />
        <BooleanInput source="published" label="Published" />
        <MarkdownInput source="text" label="Content (Markdown)" />
        <BooleanInput
          source="showCoverInPost"
          label="Show cover image inside the post"
        />
        <div>
          <label>
            <input
              type="radio"
              value="upload"
              checked={selectedOption === "upload"}
              onChange={handleChange}
            />
            Upload new cover image
          </label>
          <label style={{ marginLeft: "1em" }}>
            <input
              type="radio"
              value="link"
              checked={selectedOption === "link"}
              onChange={handleChange}
            />
            Enter link to image
          </label>
        </div>
        {selectedOption === "upload" && (
          <ImageInput
            label="Cover image"
            source="file"
            accept={IMAGE_INPUT_ACCEPT}
            validate={validateImageUpload}
          >
            <ImageField source="src" title="title" />
          </ImageInput>
        )}
        {selectedOption === "link" && (
          <TextInput label="Image URL" source="imageUrl" fullWidth />
        )}
        <div style={{ marginBottom: "1em" }}>
          <p>Current cover image:</p>
          <ImageField source="imageUrl" label="Current cover image" />
        </div>
      </SimpleForm>
    </Edit>
  );
};
