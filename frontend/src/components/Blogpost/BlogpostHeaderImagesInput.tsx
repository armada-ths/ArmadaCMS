import {
  ArrayInput,
  ImageField,
  ImageInput,
  SimpleFormIterator,
  TextInput,
  useRecordContext,
} from "react-admin";
import { Typography } from "@mui/material";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

export const BlogpostHeaderImagesInput = () => {
  const record = useRecordContext();
  return (
    <>
      <Typography variant="body2">
        Extra photos appear after the cover in the post header. Add a URL or
        upload a file for each photo. Use the arrows to reorder photos, or
        remove a row to remove a photo. A new upload replaces the URL in that
        row.
      </Typography>
      <ArrayInput
        source="headerImages"
        label="Additional header photos"
        defaultValue={(record?.imageUrls ?? []).map((url: string) => ({ url }))}
      >
        <SimpleFormIterator fullWidth>
          <TextInput source="url" label="Image URL" fullWidth />
          <ImageField source="url" label="Saved photo" />
          <ImageInput
            source="file"
            label="Upload photo"
            accept={IMAGE_INPUT_ACCEPT}
            validate={validateImageUpload}
          >
            <ImageField source="src" title="title" />
          </ImageInput>
        </SimpleFormIterator>
      </ArrayInput>
    </>
  );
};
