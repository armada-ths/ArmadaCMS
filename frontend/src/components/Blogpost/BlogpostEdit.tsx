import { BlogpostHeaderImagesInput } from "./BlogpostHeaderImagesInput";
import {
  Edit,
  EditProps,
  SimpleForm,
  TextInput,
  BooleanInput,
} from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";

export const BlogpostEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <SimpleForm>
        <TextInput source="title" fullWidth />
        <TextInput
          source="titleSv"
          label="Title (Swedish)"
          helperText="Swedish translation of the title (optional)"
          fullWidth
        />
        <TextInput source="author" fullWidth />
        <BooleanInput source="published" label="Published" />
        <MarkdownInput source="text" label="Content (Markdown)" />
        <MarkdownInput
          source="textSv"
          label="Content (Swedish - Markdown)"
          helperText="Swedish translation of the content (optional)"
        />
        <BooleanInput
          source="showCoverInPost"
          label="Show header photos inside the post"
        />
        <BlogpostHeaderImagesInput />
      </SimpleForm>
    </Edit>
  );
};
