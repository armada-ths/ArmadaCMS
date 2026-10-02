-- A durable count is needed because moderation and permanent photo deletion
-- must not return previously used upload slots to a guest.
create table public.photo_guest_uploads (
  event_id bigint not null references public.photo_events(id) on delete cascade,
  guest_hash text not null,
  upload_count bigint not null check (upload_count >= 0),
  primary key (event_id, guest_hash)
);

-- Count all existing uploads, including rejected photos.
insert into public.photo_guest_uploads (event_id, guest_hash, upload_count)
select event_id, guest_hash, count(*)
from public.event_photos
group by event_id, guest_hash;

alter table public.photo_guest_uploads enable row level security;
revoke all on public.photo_guest_uploads from anon, authenticated;

-- Quota reads now include rejected rows, so the old partial index cannot serve them.
drop index if exists public.event_photos_guest_idx;
create index event_photos_guest_idx on public.event_photos(event_id, guest_hash);
