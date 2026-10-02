import { assertValidImageUpload } from "./imageUploadValidation";

export type HeaderImageInput = {
  url?: string;
  file?: { rawFile?: File };
};

/** Preserve row order when mixing saved photos, links and new uploads. */
export const appendBlogpostHeaderImages = (
  formData: FormData,
  images: HeaderImageInput[],
) => {
  const manifest = images.map((image, index) => {
    const rawFile = image.file?.rawFile;
    if (rawFile instanceof File) {
      assertValidImageUpload(image.file);
      const field = `headerImage${index}`;
      formData.append(field, rawFile);
      return { file: field };
    }
    const url = image.url?.trim() ?? "";
    if (
      !URL.canParse(url) ||
      !["http:", "https:"].includes(new URL(url).protocol)
    ) {
      throw new Error(
        `Header photo ${index + 1}: upload an image or enter an http(s) URL.`,
      );
    }
    return { url };
  });
  formData.append("headerImages", JSON.stringify(manifest));
};
