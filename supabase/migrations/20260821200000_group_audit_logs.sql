alter table public.audit_logs
    add column if not exists parent_id bigint,
    add column if not exists group_status text not null default '',
    add column if not exists child_count integer not null default 0;

create index if not exists idx_audit_logs_parent_id
    on public.audit_logs (parent_id);

create index if not exists idx_audit_logs_group_status
    on public.audit_logs (group_status);

alter table public.audit_logs
    add constraint audit_logs_parent_id_fkey
    foreign key (parent_id)
    references public.audit_logs(id)
    on delete cascade
    not valid;

alter table public.audit_logs
    validate constraint audit_logs_parent_id_fkey;
