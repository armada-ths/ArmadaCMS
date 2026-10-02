import { useId, useState } from "react";
import {
  ArrayInput,
  ImageField,
  ImageInput,
  SimpleFormIterator,
  TextInput,
  type RaRecord,
  useRecordContext,
  useSourceContext,
  useSimpleFormIteratorItem,
} from "react-admin";
import { useFormContext, useWatch } from "react-hook-form";
import { useTheme } from "@mui/material/styles";
import { getBlogpostImages } from "@/utils/blogpostHeaderImages";
import {
  IMAGE_INPUT_ACCEPT,
  validateImageUpload,
} from "@/utils/imageUploadValidation";

const HeaderImageInputs = () => {
  const radioName = useId();
  const { getSource } = useSourceContext();
  const { setValue } = useFormContext();
  const url = useWatch({ name: getSource("url") });
  const { index } = useSimpleFormIteratorItem();
  const showHeaderPhotos = useWatch({ name: "showCoverInPost" }) !== false;
  const [option, setSelectedOption] = useState<"upload" | "link">("upload");

  const selectOption = (nextOption: "upload" | "link") => {
    if (nextOption === "link") {
      setValue(getSource("file"), null, {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
    setSelectedOption(nextOption);
  };

  return (
    <>
      <div>
        <label>
          <input
            type="radio"
            name={radioName}
            value="upload"
            checked={option === "upload"}
            onChange={() => selectOption("upload")}
          />
          Upload image
        </label>
        <label style={{ marginLeft: "1em" }}>
          <input
            type="radio"
            name={radioName}
            value="link"
            checked={option === "link"}
            onChange={() => selectOption("link")}
          />
          Enter link to existing image
        </label>
      </div>
      {option === "upload" ? (
        <ImageInput
          source="file"
          label="Upload photo"
          accept={IMAGE_INPUT_ACCEPT}
          validate={
            showHeaderPhotos || index === 0 ? validateImageUpload : undefined
          }
        >
          <ImageField source="src" title="title" />
        </ImageInput>
      ) : (
        <TextInput source="url" label="Image URL" fullWidth />
      )}
      {url && (
        <ImageField
          source="url"
          record={{ id: radioName, url }}
          label="Saved photo"
          title="Saved photo"
        />
      )}
    </>
  );
};

export const BlogpostHeaderImagesInput = () => {
  const record = useRecordContext<
    RaRecord & { imageUrl?: string | null; imageUrls?: string[] }
  >();
  const theme = useTheme();
  const showHeaderPhotos = useWatch({ name: "showCoverInPost" }) !== false;
  const images = useWatch({ name: "blogpostImages" });
  return (
    <>
      <ArrayInput
        source="blogpostImages"
        label="Blog post images"
        defaultValue={getBlogpostImages(record)}
      >
        <SimpleFormIterator
          fullWidth
          disableAdd={!showHeaderPhotos && images?.length > 0}
          disableReordering={!showHeaderPhotos}
          disableClear={!showHeaderPhotos}
          sx={{
            mt: 1,
            ...(!showHeaderPhotos && {
              "& .RaSimpleFormIterator-line:not(:first-of-type)": {
                display: "none",
              },
            }),
            "& .RaSimpleFormIterator-line + .RaSimpleFormIterator-line": {
              pt: 1,
            },
          }}
        >
          <HeaderImageInputs />
        </SimpleFormIterator>
      </ArrayInput>
      <p
        style={{
          fontSize: "0.75rem",
          color: theme.palette.text.secondary,
          marginTop: "0.25em",
        }}
      >
        {showHeaderPhotos
          ? "The first image is the cover image."
          : "Only the cover image is used while header photos are hidden. Turn the setting back on to restore the extra images before saving."}
      </p>
    </>
  );
};
