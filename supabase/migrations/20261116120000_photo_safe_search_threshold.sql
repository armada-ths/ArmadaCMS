alter table public.photo_events
  add column safe_search_max_likelihood text not null default 'UNLIKELY'
    check (safe_search_max_likelihood in ('VERY_UNLIKELY', 'UNLIKELY', 'POSSIBLE')),
  add column auto_approve_all_photos boolean not null default false,
  add constraint photo_events_single_auto_approval_mode
    check (not (auto_approve_all_photos and auto_approve_safe_photos));

alter table public.event_photos
  drop constraint event_photos_moderation_source_check,
  add constraint event_photos_moderation_source_check
    check (moderation_source in ('manual', 'vision_safe_search', 'automatic_without_safe_search'));
