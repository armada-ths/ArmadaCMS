alter table public.event_photos
  add column rejected_delete_at timestamptz;

-- Legacy rejected records have no file and remain permanently deleted.
alter table public.event_photos
  drop constraint event_photos_rejected_key,
  add constraint event_photos_rejection_retention check (
    (status = 'rejected' or rejected_delete_at is null)
    and (status <> 'rejected' or object_key is null or rejected_delete_at is not null)
  );

create index event_photos_rejected_cleanup_idx
  on public.event_photos (rejected_delete_at, id)
  where status = 'rejected' and object_key is not null;
