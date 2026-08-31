-- Preserve existing sessions while replacing reusable refresh-token values
-- with the same SHA-256 representation now used by the application.
-- Generated tokens are 64 random bytes encoded as 128 lowercase hex chars;
-- the predicate makes this migration safe to run more than once.
update public.refresh_tokens
set refresh_token = encode(extensions.digest(refresh_token, 'sha256'), 'hex')
where refresh_token ~ '^[0-9a-f]{128}$';
