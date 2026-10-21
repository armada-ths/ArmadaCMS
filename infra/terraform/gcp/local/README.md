# Local reCAPTCHA infrastructure

This root owns a score-1.0 reCAPTCHA Enterprise test key and a separate assessment API key restricted to the reCAPTCHA API. It uses the `THS-Armada/armadacms-gcp-local` workspace with local execution and remote state. It does not manage hosted keys, API enablement, IAM or Cloud Run services.

The test key permits changing developer LAN addresses. The local backend still checks token validity, `photo_upload`, score and its configured hostnames. Production and staging retain their existing keys and workload identity.

`scripts/configure-local-recaptcha.ps1` consumes the sensitive `local_recaptcha_config` output and updates only the gitignored local backend and photo-app environment files. The API key never goes to the browser. See [local development](../../../../scripts/README.md#local-development).
