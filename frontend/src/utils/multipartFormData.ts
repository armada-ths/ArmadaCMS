import { appendBlogpostHeaderImages } from "./blogpostHeaderImages";
import {
  assertValidImageUpload,
  getRawFileFromValue,
} from "./imageUploadValidation";

type ClearableField = {
  emptyValue: string;
  clearEmptyArray?: boolean;
};
type MultipartRules = {
  clearableFields?: Record<string, ClearableField>;
  blogpostImages?: boolean;
};

// Only fields whose API supports explicit clearing belong in this map.
const multipartResources: Record<string, MultipartRules> = {
  profiles: { clearableFields: { team_id: { emptyValue: "" } } },
  events: {},
  exhibitors: {},
  blogposts: { blogpostImages: true },
  "timeline-entries": {
    clearableFields: { imageUrl: { emptyValue: "", clearEmptyArray: true } },
  },
};

export const isMultipartResource = (resource: string): boolean =>
  Object.prototype.hasOwnProperty.call(multipartResources, resource);

/** Omitted/undefined fields preserve values; configured null/[] values clear them. */
export const createMultipartFormData = (
  resource: string,
  data: Record<string, unknown>,
): FormData => {
  const rules = multipartResources[resource];
  if (!isMultipartResource(resource)) {
    throw new Error(`Resource ${resource} does not use multipart requests.`);
  }
  const formData = new FormData();
  const hasCombinedImages =
    rules.blogpostImages && Array.isArray(data.blogpostImages);

  for (const [key, value] of Object.entries(data)) {
    if (value === undefined) continue;

    if (hasCombinedImages) {
      // The combined list owns the cover/gallery; stale persisted fields must not compete.
      if (
        ["imageUrl", "imageUrls", "imageFile", "file", "headerImages"].includes(
          key,
        )
      ) {
        continue;
      }
      if (key === "blogpostImages" && Array.isArray(value)) {
        appendBlogpostHeaderImages(
          formData,
          value,
          true,
          data.showCoverInPost !== false,
        );
        continue;
      }
    }
    if (
      rules.blogpostImages &&
      key === "headerImages" &&
      Array.isArray(value)
    ) {
      appendBlogpostHeaderImages(formData, value);
      continue;
    }

    const clearRule = rules.clearableFields?.[key];
    if (
      clearRule &&
      (value === null ||
        (clearRule.clearEmptyArray &&
          Array.isArray(value) &&
          value.length === 0))
    ) {
      formData.append(key, clearRule.emptyValue);
      continue;
    }
    if (value === null) continue;

    assertValidImageUpload(value);
    const rawFile = getRawFileFromValue(value);
    if (rawFile) {
      formData.append("file", rawFile);
      continue;
    }

    if (Array.isArray(value)) {
      formData.append(key, JSON.stringify(value));
    } else if (typeof value !== "object") {
      formData.append(key, String(value));
    }
  }
  return formData;
};
