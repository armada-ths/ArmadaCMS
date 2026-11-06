-- AI suggestions never update published content until an editor approves them.
CREATE TABLE public.translation_jobs (
 id bigserial PRIMARY KEY, resource text NOT NULL, record_id bigint NOT NULL,
 requested_by bigint NOT NULL, status text NOT NULL CHECK (status IN ('queued','running','budget_wait','completed','failed','cancelled','stale')),
 dedup_key text NOT NULL, source_json jsonb NOT NULL, field_ids jsonb NOT NULL,
 fingerprint text NOT NULL, model text NOT NULL, prompt text NOT NULL, glossary jsonb NOT NULL,
 prompt_version text NOT NULL, glossary_version text NOT NULL, attempts integer NOT NULL DEFAULT 0,
 error text NOT NULL DEFAULT '', slot integer CHECK (slot IN (0,1)), lease_until timestamptz,
 not_before timestamptz NOT NULL DEFAULT now(), reserved_tokens bigint NOT NULL DEFAULT 0 CHECK (reserved_tokens >= 0),
 reservation_day text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz
);
CREATE UNIQUE INDEX translation_jobs_active_key ON public.translation_jobs(dedup_key) WHERE status IN ('queued','running','budget_wait');
CREATE UNIQUE INDEX translation_jobs_running_slot ON public.translation_jobs(slot) WHERE status = 'running';
CREATE INDEX translation_jobs_queue ON public.translation_jobs(status, not_before, id);
CREATE INDEX translation_jobs_record ON public.translation_jobs(resource,record_id,id DESC);
CREATE TABLE public.translation_suggestions (
 id bigserial PRIMARY KEY, resource text NOT NULL, record_id bigint NOT NULL, field_id text NOT NULL,
 job_id bigint REFERENCES public.translation_jobs(id) ON DELETE CASCADE, version integer NOT NULL DEFAULT 1,
 model text NOT NULL DEFAULT '', prompt_version text NOT NULL DEFAULT '', glossary_version text NOT NULL DEFAULT '',
 source_fingerprint text NOT NULL, target_before text, generated text NOT NULL DEFAULT '', draft text NOT NULL DEFAULT '',
 status text NOT NULL CHECK (status IN ('draft','approved','rejected','superseded')),
 warnings jsonb NOT NULL DEFAULT '[]', published_text text, reviewed_by bigint, reviewed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX translation_suggestions_record ON public.translation_suggestions(resource,record_id,field_id,id DESC);
CREATE TABLE public.translation_usage_daily (
 day text PRIMARY KEY, used bigint NOT NULL DEFAULT 0 CHECK (used>=0), reserved bigint NOT NULL DEFAULT 0 CHECK (reserved>=0)
);
ALTER TABLE public.translation_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.translation_suggestions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.translation_usage_daily ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.translation_jobs,public.translation_suggestions,public.translation_usage_daily FROM PUBLIC,anon,authenticated;
REVOKE ALL ON SEQUENCE public.translation_jobs_id_seq,public.translation_suggestions_id_seq FROM PUBLIC,anon,authenticated;

-- Clean up private drafts even when AI generation is disabled. Running jobs
-- retain their slot/reservation until completion or lease recovery.
CREATE FUNCTION public.translation_delete_resource() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM public.translation_suggestions WHERE resource = TG_ARGV[0] AND record_id = OLD.id;
 DELETE FROM public.translation_jobs WHERE resource = TG_ARGV[0] AND record_id = OLD.id AND status <> 'running';
 RETURN OLD;
END;
$$;
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.highlight_cards FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('highlightcards');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.blogposts FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('blogposts');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.events FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('events');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.exhibitors FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('exhibitors');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.employments FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('employments');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.industries FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('industries');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.programs FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('programs');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.teams FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('teams');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.profiles FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('profiles');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.recruitment_periods FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('recruitmentperiods');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.recruitment_roles FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('recruitmentroles');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.fair_date_configs FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('fairdates');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.timeline_eras FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('timeline-eras');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.timeline_entries FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('timeline-entries');
CREATE TRIGGER translation_cleanup AFTER DELETE ON public.photo_events FOR EACH ROW EXECUTE FUNCTION public.translation_delete_resource('photoevents');
