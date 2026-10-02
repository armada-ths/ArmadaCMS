import { assertValidImageUpload } from "./imageUploadValidation";

export type HeaderImageInput = {
  url?: string;
  file?: { rawFile?: File };
};

/** Preserve row order when mixing saved photos, links and new uploads. */
export const appendBlogpostHeaderImages = (
  formData: FormData,
  images: HeaderImageInput[],
  includeCover = false,
  showHeaderPhotos = true,
) => {
  const submittedImages =
    includeCover && !showHeaderPhotos ? images.slice(0, 1) : images;
  const manifest = submittedImages.map((image, index) => {
    const rawFile = image.file?.rawFile;
    if (rawFile instanceof File) {
      assertValidImageUpload(image.file);
      const field =
        includeCover && index === 0 ? "file" : `headerImage${index}`;
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
  if (includeCover) {
    const cover = manifest.shift();
    formData.append(
      "imageUrl",
      cover && "url" in cover ? (cover.url ?? "") : "",
    );
  }
  formData.append("headerImages", JSON.stringify(manifest));
};

/** Combine the persisted cover and gallery into one sortable form list. */
export const getBlogpostImages = (record?: {
  imageUrl?: string | null;
  imageUrls?: string[];
}): HeaderImageInput[] =>
  [record?.imageUrl, ...(record?.imageUrls ?? [])]
    .filter((url): url is string => !!url)
    .map((url) => ({ url }));
