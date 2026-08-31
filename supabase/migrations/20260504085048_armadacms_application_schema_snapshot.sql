create sequence "public"."audit_logs_id_seq";

create sequence "public"."blogposts_id_seq";

create sequence "public"."employments_id_seq";

create sequence "public"."events_id_seq";

create sequence "public"."exhibitors_id_seq";

create sequence "public"."fair_date_configs_id_seq";

create sequence "public"."feature_flags_id_seq";

create sequence "public"."highlight_cards_id_seq";

create sequence "public"."industries_id_seq";

create sequence "public"."profiles_id_seq";

create sequence "public"."programs_id_seq";

create sequence "public"."recruitment_periods_id_seq";

create sequence "public"."recruitment_roles_id_seq";

create sequence "public"."refresh_tokens_id_seq";

create sequence "public"."roles_id_seq";

create sequence "public"."teams_id_seq";

create sequence "public"."users_id_seq";


  create table "public"."audit_logs" (
    "id" bigint not null default nextval('public.audit_logs_id_seq'::regclass),
    "actor_user_id" bigint,
    "actor_username" text,
    "actor_name" text,
    "action" text not null,
    "resource_type" text not null,
    "resource_id" text not null,
    "request_path" text not null,
    "http_method" text not null,
    "old_data" jsonb not null default 'null'::jsonb,
    "new_data" jsonb not null default 'null'::jsonb,
    "created_at" timestamp with time zone default now()
      );



  create table "public"."blogposts" (
    "id" bigint not null default nextval('public.blogposts_id_seq'::regclass),
    "user_id" bigint not null,
    "text" text,
    "title" text,
    "author" text,
    "image_url" text,
    "show_cover_in_post" boolean default true,
    "created_at" timestamp with time zone default CURRENT_TIMESTAMP
      );



  create table "public"."employments" (
    "id" bigint not null default nextval('public.employments_id_seq'::regclass),
    "name" text not null
      );



  create table "public"."events" (
    "id" bigint not null default nextval('public.events_id_seq'::regclass),
    "eventro_id" text,
    "name" text not null,
    "description" text,
    "location" text not null,
    "food" text,
    "event_start" timestamp with time zone not null,
    "event_end" timestamp with time zone not null,
    "registration_end" timestamp with time zone,
    "image_url" text,
    "fee" text,
    "registration_required" boolean not null,
    "signup_link" text,
    "event_max_capacity" bigint,
    "show" boolean not null default true
      );



  create table "public"."exhibitor_employments" (
    "exhibitor_id" bigint not null,
    "employment_id" bigint not null
      );



  create table "public"."exhibitor_industries" (
    "exhibitor_id" bigint not null,
    "industry_id" bigint not null
      );



  create table "public"."exhibitor_programs" (
    "exhibitor_id" bigint not null,
    "program_id" bigint not null
      );



  create table "public"."exhibitors" (
    "id" bigint not null default nextval('public.exhibitors_id_seq'::regclass),
    "eventro_id" text,
    "name" text not null,
    "type" text not null,
    "tier" text,
    "company_website" text,
    "about" text,
    "purpose" text,
    "logo_squared_url" text,
    "logo_freesize_url" text,
    "map_img" text,
    "cities" text,
    "fair_location" text not null,
    "vyer_position" text,
    "location_special" text,
    "climate_compensation" boolean not null,
    "flyer" text not null
      );



  create table "public"."fair_date_configs" (
    "id" bigint not null default nextval('public.fair_date_configs_id_seq'::regclass),
    "description" text not null,
    "fair_days" text not null,
    "ir_start" text not null,
    "ir_end" text not null,
    "ir_acceptance" text not null,
    "fr_start" text not null,
    "fr_end" text not null,
    "events_start" text not null
      );



  create table "public"."feature_flags" (
    "id" bigint not null default nextval('public.feature_flags_id_seq'::regclass),
    "key" text not null,
    "description" text not null,
    "enabled" boolean not null
      );



  create table "public"."highlight_cards" (
    "id" bigint not null default nextval('public.highlight_cards_id_seq'::regclass),
    "title" text not null,
    "subtitle" text not null,
    "description" text not null,
    "brand" text default 'ARMADA'::text,
    "link_text" text,
    "link_url" text,
    "cta_event_name" text
      );



  create table "public"."industries" (
    "id" bigint not null default nextval('public.industries_id_seq'::regclass),
    "name" text not null
      );



  create table "public"."profiles" (
    "id" bigint not null default nextval('public.profiles_id_seq'::regclass),
    "eventro_key" text,
    "name" text not null,
    "team_id" bigint,
    "linkedin" text,
    "email" text,
    "photo" text,
    "rank" text,
    "title" text
      );



  create table "public"."programs" (
    "id" bigint not null default nextval('public.programs_id_seq'::regclass),
    "name" text not null
      );



  create table "public"."recruitment_periods" (
    "id" bigint not null default nextval('public.recruitment_periods_id_seq'::regclass),
    "eventro_id" text,
    "name" text not null default ''::text,
    "link" text not null default ''::text,
    "start_date" timestamp with time zone,
    "end_date" timestamp with time zone
      );



  create table "public"."recruitment_roles" (
    "id" bigint not null default nextval('public.recruitment_roles_id_seq'::regclass),
    "eventro_role_id" text,
    "recruitment_id" bigint not null,
    "team_id" bigint,
    "name" text not null default ''::text,
    "description" text not null default ''::text
      );



  create table "public"."refresh_tokens" (
    "id" bigint not null default nextval('public.refresh_tokens_id_seq'::regclass),
    "refresh_token" text not null,
    "valid_from" timestamp with time zone not null,
    "valid_to" timestamp with time zone not null,
    "user_id" bigint not null,
    "enabled" boolean default true
      );



  create table "public"."roles" (
    "id" bigint not null default nextval('public.roles_id_seq'::regclass),
    "name" text not null,
    "permissions" text not null default '[]'::text
      );



  create table "public"."teams" (
    "id" bigint not null default nextval('public.teams_id_seq'::regclass),
    "team_name" text not null
      );



  create table "public"."users" (
    "id" bigint not null default nextval('public.users_id_seq'::regclass),
    "username" text not null,
    "password" text not null,
    "name" text not null,
    "avatar" text,
    "role_id" bigint,
    "created_at" timestamp with time zone default now(),
    "updated_at" timestamp with time zone default now()
      );


alter sequence "public"."audit_logs_id_seq" owned by "public"."audit_logs"."id";

alter sequence "public"."blogposts_id_seq" owned by "public"."blogposts"."id";

alter sequence "public"."employments_id_seq" owned by "public"."employments"."id";

alter sequence "public"."events_id_seq" owned by "public"."events"."id";

alter sequence "public"."exhibitors_id_seq" owned by "public"."exhibitors"."id";

alter sequence "public"."fair_date_configs_id_seq" owned by "public"."fair_date_configs"."id";

alter sequence "public"."feature_flags_id_seq" owned by "public"."feature_flags"."id";

alter sequence "public"."highlight_cards_id_seq" owned by "public"."highlight_cards"."id";

alter sequence "public"."industries_id_seq" owned by "public"."industries"."id";

alter sequence "public"."profiles_id_seq" owned by "public"."profiles"."id";

alter sequence "public"."programs_id_seq" owned by "public"."programs"."id";

alter sequence "public"."recruitment_periods_id_seq" owned by "public"."recruitment_periods"."id";

alter sequence "public"."recruitment_roles_id_seq" owned by "public"."recruitment_roles"."id";

alter sequence "public"."refresh_tokens_id_seq" owned by "public"."refresh_tokens"."id";

alter sequence "public"."roles_id_seq" owned by "public"."roles"."id";

alter sequence "public"."teams_id_seq" owned by "public"."teams"."id";

alter sequence "public"."users_id_seq" owned by "public"."users"."id";

CREATE UNIQUE INDEX audit_logs_pkey ON public.audit_logs USING btree (id);

CREATE UNIQUE INDEX blogposts_pkey ON public.blogposts USING btree (id);

CREATE UNIQUE INDEX employments_pkey ON public.employments USING btree (id);

CREATE UNIQUE INDEX events_pkey ON public.events USING btree (id);

CREATE UNIQUE INDEX exhibitor_employments_pkey ON public.exhibitor_employments USING btree (exhibitor_id, employment_id);

CREATE UNIQUE INDEX exhibitor_industries_pkey ON public.exhibitor_industries USING btree (exhibitor_id, industry_id);

CREATE UNIQUE INDEX exhibitor_programs_pkey ON public.exhibitor_programs USING btree (exhibitor_id, program_id);

CREATE UNIQUE INDEX exhibitors_pkey ON public.exhibitors USING btree (id);

CREATE UNIQUE INDEX fair_date_configs_pkey ON public.fair_date_configs USING btree (id);

CREATE UNIQUE INDEX feature_flags_pkey ON public.feature_flags USING btree (id);

CREATE UNIQUE INDEX highlight_cards_pkey ON public.highlight_cards USING btree (id);

CREATE INDEX idx_audit_logs_action ON public.audit_logs USING btree (action);

CREATE INDEX idx_audit_logs_actor_user_id ON public.audit_logs USING btree (actor_user_id);

CREATE INDEX idx_audit_logs_created_at ON public.audit_logs USING btree (created_at);

CREATE INDEX idx_audit_logs_resource_id ON public.audit_logs USING btree (resource_id);

CREATE INDEX idx_audit_logs_resource_type ON public.audit_logs USING btree (resource_type);

CREATE UNIQUE INDEX idx_employments_name ON public.employments USING btree (name);

CREATE UNIQUE INDEX idx_events_eventro_id ON public.events USING btree (eventro_id);

CREATE UNIQUE INDEX idx_exhibitors_eventro_id ON public.exhibitors USING btree (eventro_id);

CREATE UNIQUE INDEX idx_feature_flags_key ON public.feature_flags USING btree (key);

CREATE UNIQUE INDEX idx_industries_name ON public.industries USING btree (name);

CREATE UNIQUE INDEX idx_profiles_eventro_key ON public.profiles USING btree (eventro_key);

CREATE UNIQUE INDEX idx_programs_name ON public.programs USING btree (name);

CREATE UNIQUE INDEX idx_recruitment_periods_eventro_id ON public.recruitment_periods USING btree (eventro_id);

CREATE UNIQUE INDEX idx_recruitment_roles_eventro_role_id ON public.recruitment_roles USING btree (eventro_role_id);

CREATE INDEX idx_recruitment_roles_recruitment_id ON public.recruitment_roles USING btree (recruitment_id);

CREATE INDEX idx_recruitment_roles_team_id ON public.recruitment_roles USING btree (team_id);

CREATE UNIQUE INDEX idx_roles_name ON public.roles USING btree (name);

CREATE UNIQUE INDEX industries_pkey ON public.industries USING btree (id);

CREATE UNIQUE INDEX profiles_pkey ON public.profiles USING btree (id);

CREATE UNIQUE INDEX programs_pkey ON public.programs USING btree (id);

CREATE UNIQUE INDEX recruitment_periods_pkey ON public.recruitment_periods USING btree (id);

CREATE UNIQUE INDEX recruitment_roles_pkey ON public.recruitment_roles USING btree (id);

CREATE UNIQUE INDEX refresh_tokens_pkey ON public.refresh_tokens USING btree (id);

CREATE UNIQUE INDEX roles_pkey ON public.roles USING btree (id);

CREATE UNIQUE INDEX teams_pkey ON public.teams USING btree (id);

CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id);

alter table "public"."audit_logs" add constraint "audit_logs_pkey" PRIMARY KEY using index "audit_logs_pkey";

alter table "public"."blogposts" add constraint "blogposts_pkey" PRIMARY KEY using index "blogposts_pkey";

alter table "public"."employments" add constraint "employments_pkey" PRIMARY KEY using index "employments_pkey";

alter table "public"."events" add constraint "events_pkey" PRIMARY KEY using index "events_pkey";

alter table "public"."exhibitor_employments" add constraint "exhibitor_employments_pkey" PRIMARY KEY using index "exhibitor_employments_pkey";

alter table "public"."exhibitor_industries" add constraint "exhibitor_industries_pkey" PRIMARY KEY using index "exhibitor_industries_pkey";

alter table "public"."exhibitor_programs" add constraint "exhibitor_programs_pkey" PRIMARY KEY using index "exhibitor_programs_pkey";

alter table "public"."exhibitors" add constraint "exhibitors_pkey" PRIMARY KEY using index "exhibitors_pkey";

alter table "public"."fair_date_configs" add constraint "fair_date_configs_pkey" PRIMARY KEY using index "fair_date_configs_pkey";

alter table "public"."feature_flags" add constraint "feature_flags_pkey" PRIMARY KEY using index "feature_flags_pkey";

alter table "public"."highlight_cards" add constraint "highlight_cards_pkey" PRIMARY KEY using index "highlight_cards_pkey";

alter table "public"."industries" add constraint "industries_pkey" PRIMARY KEY using index "industries_pkey";

alter table "public"."profiles" add constraint "profiles_pkey" PRIMARY KEY using index "profiles_pkey";

alter table "public"."programs" add constraint "programs_pkey" PRIMARY KEY using index "programs_pkey";

alter table "public"."recruitment_periods" add constraint "recruitment_periods_pkey" PRIMARY KEY using index "recruitment_periods_pkey";

alter table "public"."recruitment_roles" add constraint "recruitment_roles_pkey" PRIMARY KEY using index "recruitment_roles_pkey";

alter table "public"."refresh_tokens" add constraint "refresh_tokens_pkey" PRIMARY KEY using index "refresh_tokens_pkey";

alter table "public"."roles" add constraint "roles_pkey" PRIMARY KEY using index "roles_pkey";

alter table "public"."teams" add constraint "teams_pkey" PRIMARY KEY using index "teams_pkey";

alter table "public"."users" add constraint "users_pkey" PRIMARY KEY using index "users_pkey";

alter table "public"."exhibitor_employments" add constraint "fk_exhibitor_employments_employment" FOREIGN KEY (employment_id) REFERENCES public.employments(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_employments" validate constraint "fk_exhibitor_employments_employment";

alter table "public"."exhibitor_employments" add constraint "fk_exhibitor_employments_exhibitor" FOREIGN KEY (exhibitor_id) REFERENCES public.exhibitors(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_employments" validate constraint "fk_exhibitor_employments_exhibitor";

alter table "public"."exhibitor_industries" add constraint "fk_exhibitor_industries_exhibitor" FOREIGN KEY (exhibitor_id) REFERENCES public.exhibitors(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_industries" validate constraint "fk_exhibitor_industries_exhibitor";

alter table "public"."exhibitor_industries" add constraint "fk_exhibitor_industries_industry" FOREIGN KEY (industry_id) REFERENCES public.industries(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_industries" validate constraint "fk_exhibitor_industries_industry";

alter table "public"."exhibitor_programs" add constraint "fk_exhibitor_programs_exhibitor" FOREIGN KEY (exhibitor_id) REFERENCES public.exhibitors(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_programs" validate constraint "fk_exhibitor_programs_exhibitor";

alter table "public"."exhibitor_programs" add constraint "fk_exhibitor_programs_program" FOREIGN KEY (program_id) REFERENCES public.programs(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."exhibitor_programs" validate constraint "fk_exhibitor_programs_program";

alter table "public"."profiles" add constraint "fk_profiles_team" FOREIGN KEY (team_id) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE SET NULL not valid;

alter table "public"."profiles" validate constraint "fk_profiles_team";

alter table "public"."recruitment_roles" add constraint "fk_recruitment_periods_roles" FOREIGN KEY (recruitment_id) REFERENCES public.recruitment_periods(id) ON UPDATE CASCADE ON DELETE CASCADE not valid;

alter table "public"."recruitment_roles" validate constraint "fk_recruitment_periods_roles";

alter table "public"."recruitment_roles" add constraint "fk_recruitment_roles_team" FOREIGN KEY (team_id) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE SET NULL not valid;

alter table "public"."recruitment_roles" validate constraint "fk_recruitment_roles_team";

alter table "public"."refresh_tokens" add constraint "fk_refresh_tokens_user" FOREIGN KEY (user_id) REFERENCES public.users(id) not valid;

alter table "public"."refresh_tokens" validate constraint "fk_refresh_tokens_user";

alter table "public"."users" add constraint "fk_users_role" FOREIGN KEY (role_id) REFERENCES public.roles(id) ON UPDATE CASCADE ON DELETE SET NULL not valid;

alter table "public"."users" validate constraint "fk_users_role";
