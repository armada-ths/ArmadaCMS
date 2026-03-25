export const IMAGE_INPUT_ACCEPT = {
  "image/jpeg": [],
  "image/png": [],
  "image/webp": [],
  "image/gif": [],
};

const ALLOWED_IMAGE_MIME_TYPES = new Set(Object.keys(IMAGE_INPUT_ACCEPT));

const ALLOWED_IMAGE_EXTENSIONS = new Set([
  ".jpg",
  ".jpeg",
  ".png",
  ".webp",
  ".gif",
]);

const UNSUPPORTED_IMAGE_ERROR =
  "Unsupported file format. Please upload a JPG, PNG, WEBP, or GIF image.";

const getExtension = (fileName: string) => {
  const dotIndex = fileName.lastIndexOf(".");
  return dotIndex >= 0 ? fileName.slice(dotIndex).toLowerCase() : "";
};

const getRawFileFromValue = (value: unknown): File | null => {
  if (value == null) return null;

  if (Array.isArray(value)) {
    const fileEntry = value.find(
      (
        entry,
      ): entry is {
        rawFile?: File;
      } =>
        typeof entry === "object" &&
        entry !== null &&
        "rawFile" in entry &&
        (entry as { rawFile?: File }).rawFile instanceof File,
    );

    return fileEntry?.rawFile ?? null;
  }

  if (
    typeof value === "object" &&
    value !== null &&
    "rawFile" in value &&
    (value as { rawFile?: File }).rawFile instanceof File
  ) {
    return (value as { rawFile: File }).rawFile;
  }

  return null;
};

export const validateImageUpload = (value: unknown) => {
  const file = getRawFileFromValue(value);
  if (!file) return undefined;

  const mimeType = file.type.toLowerCase();
  if (mimeType && ALLOWED_IMAGE_MIME_TYPES.has(mimeType)) {
    return undefined;
  }

  const extension = getExtension(file.name);
  if (!mimeType && extension && ALLOWED_IMAGE_EXTENSIONS.has(extension)) {
    return undefined;
  }

  return UNSUPPORTED_IMAGE_ERROR;
};

export const assertValidImageUpload = (value: unknown) => {
  const validationResult = validateImageUpload(value);
  if (validationResult) {
    throw new Error(validationResult);
  }
};
