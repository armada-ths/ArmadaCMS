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
  useSaveContext,
} from "react-admin";
import { Button } from "@mui/material";
import SaveIcon from "@mui/icons-material/Save";
import { useFormContext } from "react-hook-form";
import { MarkdownInput } from "../shared/MarkdownInput";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

const BlogpostCreateToolbar = () => {
  const { save } = useSaveContext();
  const { handleSubmit, setValue } = useFormContext();

  const submitWith = (published: boolean) => {
    setValue("published", published);
    void handleSubmit((values) => save?.(values))();
  };

  return (
    <Toolbar sx={{ gap: 1 }}>
      <Button
        type="button"
        variant="contained"
        onClick={() => submitWith(true)}
        startIcon={<SaveIcon />}
      >
        Publish
      </Button>
      <Button
        type="button"
        variant="outlined"
        onClick={() => submitWith(false)}
        startIcon={<SaveIcon />}
      >
        Save as draft
      </Button>
    </Toolbar>
  );
};

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
        {/* Registered but hidden — value is controlled by the toolbar buttons. */}
        <BooleanInput
          source="published"
          defaultValue={true}
          sx={{ display: "none" }}
        />
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
