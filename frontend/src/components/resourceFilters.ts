import type { ResourceName } from "../navigation/catalog";

export type ResourceFilter = {
  source: string;
  label: string;
  type?: "text" | "number" | "boolean" | "date" | "datetime" | "select";
  choices?: { id: string; name: string }[];
};
const text = (source: string, label: string): ResourceFilter => ({
  source,
  label,
});
const number = (source: string, label: string): ResourceFilter => ({
  source,
  label,
  type: "number",
});
const boolean = (source: string, label: string): ResourceFilter => ({
  source,
  label,
  type: "boolean",
});
const date = (source: string, label: string): ResourceFilter => ({
  source,
  label,
  type: "date",
});
const datetime = (source: string, label: string): ResourceFilter => ({
  source,
  label,
  type: "datetime",
});
const id = number("id", "ID");
const name = text("name", "Name");
const description = text("description", "Description");
const title = text("title", "Title");
const eventroId = text("eventro_id", "Eventro ID");
const named = [id, name];

/** Exact column filters. Sources use database names; the API also accepts JSON aliases. */
export const RESOURCE_FILTERS: Record<ResourceName, ResourceFilter[]> = {
  programs: named,
  industries: named,
  employments: named,
  teams: [id, text("team_name", "Team name")],
  roles: named,
  customusers: [
    id,
    text("username", "Username"),
    name,
    text("avatar", "Avatar URL"),
    datetime("created_at", "Created at"),
    datetime("updated_at", "Updated at"),
  ],
  profiles: [
    id,
    name,
    text("eventro_key", "Eventro key"),
    number("team_id", "Team ID"),
    text("title", "Title"),
    text("email", "Email"),
    text("rank", "Rank"),
    text("linkedin", "LinkedIn URL"),
    text("photo", "Photo URL"),
  ],
  events: [
    id,
    name,
    eventroId,
    description,
    text("location", "Location"),
    text("open_to", "Open to"),
    text("food", "Food"),
    text("fee", "Fee"),
    boolean("show", "Show"),
    boolean("registration_required", "Registration required"),
    datetime("event_start", "Event start"),
    datetime("event_end", "Event end"),
    datetime("registration_end", "Registration end"),
    number("event_max_capacity", "Maximum capacity"),
    text("map_url", "Map URL"),
    text("image_url", "Image URL"),
    text("signup_link", "Signup link"),
  ],
  exhibitors: [
    id,
    name,
    eventroId,
    text("type", "Type"),
    {
      source: "tier",
      label: "Tier",
      type: "select",
      choices: ["Standard", "Bronze", "Silver", "Gold"].map((tier) => ({
        id: tier,
        name: tier,
      })),
    },
    text("about", "About"),
    text("purpose", "Purpose"),
    text("company_website", "Company website"),
    text("cities", "Cities"),
    text("fair_location", "Fair location"),
    text("location_special", "Special location"),
    boolean("climate_compensation", "Climate compensation"),
    text("vyer_position", "Vyer position"),
    text("flyer", "Flyer"),
    text("logo_squared_url", "Square logo URL"),
    text("logo_freesize_url", "Free-size logo URL"),
    text("map_img", "Map image URL"),
  ],
  fairdates: [
    id,
    eventroId,
    description,
    text("fair_days", "Fair days"),
    date("ir_start", "IR start"),
    date("ir_end", "IR end"),
    date("ir_acceptance", "IR acceptance"),
    date("fr_start", "FR start"),
    date("fr_end", "FR end"),
    date("events_start", "Events start"),
  ],
  featureflags: [
    id,
    text("key", "Key"),
    description,
    boolean("enabled", "Enabled"),
  ],
  highlightcards: [
    id,
    title,
    text("subtitle", "Subtitle"),
    description,
    text("brand", "Brand"),
    text("link_text", "Link text"),
    text("link_url", "Link URL"),
    text("cta_event_name", "CTA event name"),
  ],
  blogposts: [
    id,
    title,
    text("text", "Text"),
    text("author", "Author"),
    number("user_id", "Author ID"),
    boolean("published", "Published"),
    boolean("show_cover_in_post", "Show cover in post"),
    text("image_url", "Cover image URL"),
    datetime("created_at", "Created at"),
  ],
  "timeline-entries": [
    id,
    title,
    text("body", "Body"),
    number("era_id", "Era ID"),
    number("sort_order", "Sort order"),
    text("image_url", "Image URL"),
  ],
  "timeline-eras": [id, title, number("sort_order", "Sort order")],
  recruitmentperiods: [
    id,
    name,
    eventroId,
    text("link", "Link"),
    datetime("start_date", "Start date"),
    datetime("end_date", "End date"),
  ],
  recruitmentroles: [
    id,
    name,
    description,
    text("eventro_role_id", "Eventro role ID"),
    number("recruitment_id", "Recruitment period ID"),
    number("team_id", "Team ID"),
  ],
  photoevents: [
    id,
    name,
    description,
    datetime("uploads_open_at", "Uploads open"),
    datetime("uploads_close_at", "Uploads close"),
    datetime("gallery_close_at", "Gallery closes"),
    number("max_photos_per_guest", "Photos per guest"),
    boolean("auto_approve_safe_photos", "Auto-approve safe photos"),
    datetime("deletion_requested_at", "Deletion requested at"),
    datetime("deletion_completed_at", "Deletion completed at"),
    datetime("created_at", "Created at"),
    datetime("updated_at", "Updated at"),
  ],
  // Audit logs retain their existing action, resource and username filters.
  auditlogs: [],
};
