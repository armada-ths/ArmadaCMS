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
        <TextInput source="author" fullWidth />
        <BooleanInput source="published" label="Published" />
        <MarkdownInput source="text" label="Content (Markdown)" />
        <BooleanInput
          source="showCoverInPost"
          label="Show header photos inside the post"
        />
        <BlogpostHeaderImagesInput />
      </SimpleForm>
    </Edit>
  );
};
