export type Photo = {
  id: number;
  status: string;
  thumbnail_url?: string;
  urls_expires_at?: string;
  uploaded_at: string;
  rejected_delete_at?: string | null;
  can_restore?: boolean;
  moderation_source?: "manual" | "vision_safe_search" | null;
  ai_review_status: "not_scanned" | "safe" | "review" | "error";
  ai_likelihoods?: Record<string, string> | null;
};

export const canSelectPhoto = (photo: Photo, now: number) =>
  photo.status !== "rejected" ||
  (photo.can_restore === true &&
    !!photo.rejected_delete_at &&
    new Date(photo.rejected_delete_at).getTime() > now);
