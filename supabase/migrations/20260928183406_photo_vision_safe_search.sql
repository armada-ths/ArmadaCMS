alter table public.photo_events
  add column auto_approve_safe_photos boolean not null default false;

alter table public.event_photos
  add column moderation_source text check (moderation_source in ('manual', 'vision_safe_search')),
  add column ai_review_status text not null default 'not_scanned'
    check (ai_review_status in ('not_scanned', 'safe', 'review', 'error')),
  add column ai_likelihoods jsonb,
  add column ai_checked_at timestamptz;
