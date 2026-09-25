create table public.photo_events (
  id bigint generated always as identity primary key,
  name text not null check (length(trim(name)) > 0),
  slug text not null unique check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  description text not null default '',
  uploads_open_at timestamptz not null,
  uploads_close_at timestamptz not null,
  gallery_close_at timestamptz not null,
  deletion_requested_at timestamptz,
  deletion_completed_at timestamptz,
  active boolean not null default false,
  privacy_url text not null default '',
  max_photos_per_guest integer not null default 25 check (max_photos_per_guest between 1 and 25),
  token_version integer not null default 1 check (token_version > 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint photo_events_time_order check (uploads_open_at < uploads_close_at and uploads_close_at <= gallery_close_at),
  constraint photo_events_active_privacy check (not active or length(trim(privacy_url)) > 0)
);

create index photo_events_deletion_queue_idx on public.photo_events(deletion_requested_at)
  where deletion_requested_at is not null and deletion_completed_at is null;

create table public.event_photos (
  id bigint generated always as identity primary key,
  event_id bigint not null references public.photo_events(id) on delete cascade,
  object_key text,
  status text not null default 'pending' check (status in ('pending', 'approved', 'rejected')),
  guest_hash text not null,
  byte_size bigint not null check (byte_size >= 0),
  width integer not null check (width > 0),
  height integer not null check (height > 0),
  uploaded_at timestamptz not null default now(),
  moderated_at timestamptz,
  moderated_by bigint references public.users(id),
  constraint event_photos_rejected_key check (status <> 'rejected' or object_key is null)
);

create index event_photos_event_idx on public.event_photos(event_id);
create index event_photos_pending_idx on public.event_photos(event_id, uploaded_at, id) where status = 'pending';
create index event_photos_gallery_idx on public.event_photos(event_id, moderated_at desc, id desc) where status = 'approved';
create index event_photos_guest_idx on public.event_photos(event_id, guest_hash) where status <> 'rejected';

create table public.photo_exports (
  id bigint generated always as identity primary key,
  event_id bigint not null references public.photo_events(id) on delete cascade,
  status text not null default 'queued' check (status in ('queued', 'running', 'completed', 'failed')),
  object_key text,
  requested_by bigint references public.users(id),
  created_at timestamptz not null default now(),
  started_at timestamptz,
  finished_at timestamptz,
  expires_at timestamptz,
  error text
);

create index photo_exports_event_idx on public.photo_exports(event_id);
create index photo_exports_queue_idx on public.photo_exports(created_at, id) where status in ('queued', 'running');

alter table public.photo_events enable row level security;
alter table public.event_photos enable row level security;
alter table public.photo_exports enable row level security;
revoke all on public.photo_events, public.event_photos, public.photo_exports from anon, authenticated;

insert into storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
values ('event-photos', 'event-photos', false, 26214400, array['image/jpeg'])
on conflict (id) do update set name = excluded.name, public = false,
  file_size_limit = excluded.file_size_limit, allowed_mime_types = excluded.allowed_mime_types;

insert into storage.buckets (id, name, public, allowed_mime_types)
values ('event-photo-exports', 'event-photo-exports', false, array['application/zip'])
on conflict (id) do update set name = excluded.name, public = false,
  allowed_mime_types = excluded.allowed_mime_types;
