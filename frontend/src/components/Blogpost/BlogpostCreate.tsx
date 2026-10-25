import { BlogpostHeaderImagesInput } from "./BlogpostHeaderImagesInput";
import {
  Create,
  CreateProps,
  SimpleForm,
  TextInput,
  BooleanInput,
  Toolbar,
  useSaveContext,
} from "react-admin";
import { Button } from "@mui/material";
import SaveIcon from "@mui/icons-material/Save";
import { useFormContext } from "react-hook-form";
import { MarkdownInput } from "../shared/MarkdownInput";

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
          label="Show header photos inside the post"
          defaultValue={true}
        />
        <BlogpostHeaderImagesInput />
      </SimpleForm>
    </Create>
  );
};
