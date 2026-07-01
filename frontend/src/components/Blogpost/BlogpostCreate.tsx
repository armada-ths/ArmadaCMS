import { useState } from "react";
import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  ImageInput,
  ImageField,
  BooleanInput,
  Toolbar,
  SaveButton,
} from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

const BlogpostCreateToolbar = () => (
  <Toolbar sx={{ gap: 1 }}>
    <SaveButton
      label="Publish"
      transform={(data) => ({ ...data, published: true })}
    />
    <SaveButton
      label="Save as draft"
      variant="outlined"
      transform={(data) => ({ ...data, published: false })}
    />
  </Toolbar>
);

export const BlogpostCreate = (props: CreateProps) => {
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
      <SimpleForm toolbar={<BlogpostCreateToolbar />}>
        <TextInput source="title" fullWidth />
        <TextInput source="author" fullWidth />
        <MarkdownInput source="text" label="Content (Markdown)" />
        <BooleanInput
          source="showCoverInPost"
          label="Show cover image inside the post"
          defaultValue={true}
        />
        <div>
          <label>
            <input
              type="radio"
              value="upload"
              checked={selectedOption === "upload"}
              onChange={handleChange}
            />
            Upload cover image
          </label>
          <label style={{ marginLeft: "1em" }}>
            <input
              type="radio"
              value="link"
              checked={selectedOption === "link"}
              onChange={handleChange}
            />
            Enter link to existing image
          </label>
        </div>
        {selectedOption === "upload" ? (
          <ImageInput
            label="Cover image"
            source="imageFile"
            accept={IMAGE_INPUT_ACCEPT}
            validate={validateImageUpload}
          >
            <ImageField source="src" title="title" />
          </ImageInput>
        ) : (
          <TextInput label="Image URL" source="imageUrl" fullWidth />
        )}
      </SimpleForm>
    </Create>
  );
};
