import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Title, useBasename, useBlocker } from "react-admin";
import { Link } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  FormControlLabel,
  MenuItem,
  Paper,
  Stack,
  TablePagination,
  TextField,
  Typography,
} from "@mui/material";
import { generateResources, type BulkProgress } from "./bulk";
import { MarkdownEditor } from "../shared/MarkdownInput";
import {
  translationRequest,
  type Overview,
  type Resource,
  type TranslationField,
  type TranslationPage,
  type TranslationRecord,
} from "./api";
import { useTranslationDrafts, type TranslationDraft } from "./drafts";

const message = (error: unknown) =>
  error instanceof Error ? error.message : "Translation request failed";
const keyFor = (resource: string, id: number) => `${resource}/${id}`;
const formatLabel = (id: string) => id.replace(/([a-z])([A-Z])/g, "$1 $2");
const fieldValue = (field: TranslationField) =>
  field.suggestion?.status === "draft"
    ? field.suggestion.draft
    : (field.published ?? "");
const statuses = [
  ["missing", "Missing or outdated translations"],
  ["review", "Ready for review"],
  ["needs_update", "Published translations need updating"],
  ["stale", "Outdated suggestions"],
  ["rejected", "Rejected suggestions"],
  ["queued", "Queued"],
  ["running", "Running"],
  ["budget_wait", "Waiting for budget"],
  ["failed", "Generation failed"],
  ["all", "All records"],
];
export function Translations() {
  const client = useQueryClient();
  const { drafts } = useTranslationDrafts();
  const [choice, setChoice] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("missing");
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<number[]>([]);
  const [busy, setBusy] = useState(false);
  const [selectedResources, setSelectedResources] = useState<string[]>([]);
  const [bulkProgress, setBulkProgress] = useState<BulkProgress | null>(null);
  const [bulkResult, setBulkResult] = useState("");
  const bulkAbort = useRef<AbortController | null>(null);
  useEffect(() => () => bulkAbort.current?.abort(), []);
  const [error, setError] = useState("");
  const dirty = Object.values(drafts).some(
    (d) => Object.keys(d.texts).length > 0,
  );
  const blocker = useBlocker(
    ({ nextLocation }) =>
      (dirty || busy) && !nextLocation.pathname.includes("/translations"),
  );
  useEffect(() => {
    if (blocker.state !== "blocked") return;
    if (
      window.confirm(
        "Leave translations? Unsaved edits remain available while the CMS stays open. Requests already queued will continue; further queueing will stop.",
      )
    ) {
      blocker.proceed?.();
    } else {
      blocker.reset?.();
    }
  }, [blocker]);
  const overview = useQuery({
    queryKey: ["translations", "overview"],
    queryFn: ({ signal }) =>
      translationRequest<Overview>("", undefined, "GET", signal),
  });
  const resources = overview.data?.resources ?? [];
  const eligibleResources = resources.filter((r) => r.canEdit && r.aiAllowed);
  const bulkResources = eligibleResources.filter((r) =>
    selectedResources.includes(r.id),
  );
  const resource = resources.find((r) => r.id === choice) ?? resources[0];
  const records = useQuery({
    queryKey: ["translations", "records", resource?.id, search, status, page],
    enabled: !!resource,
    queryFn: ({ signal }) =>
      translationRequest<TranslationPage>(
        `/${resource!.id}?${new URLSearchParams({ q: search, status, page: String(page), size: "10" })}`,
        undefined,
        "GET",
        signal,
      ),
    refetchInterval: 5000,
  });
  const refresh = () =>
    client.invalidateQueries({ queryKey: ["translations"] });
  async function generate(ids: number[], regenerate = false) {
    if (!resource || !ids.length) return;
    if (
      !window.confirm(
        `${regenerate ? "Regenerate" : "Generate"} suggestions for ${ids.length} selected record(s)? Existing local edits will be kept.`,
      )
    )
      return;
    setBusy(true);
    setError("");
    try {
      const jobs = await translationRequest<unknown[]>(
        `/${resource.id}/generate`,
        { ids, regenerate },
      );
      if (!jobs.length)
        setError(
          "No eligible fields. Existing manual translations are preserved. Use Regenerate explicitly for rejected or existing suggestions.",
        );
      setSelected([]);
      await refresh();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }
  async function generateBulk() {
    if (
      !bulkResources.length ||
      !window.confirm(
        `Generate suggestions across all pages of ${bulkResources.length} selected resource(s)? Search/status filters below do not apply. Existing manual translations and local edits are preserved. This does not approve any content.`,
      )
    )
      return;
    const controller = new AbortController();
    bulkAbort.current = controller;
    setBusy(true);
    setError("");
    setBulkResult("");
    try {
      const result = await generateResources(
        bulkResources,
        setBulkProgress,
        controller.signal,
      );
      setBulkResult(
        `Checked ${result.checked} records; ${result.jobs} jobs queued or already scheduled. Records without eligible fields were skipped.`,
      );
      setSelectedResources([]);
    } catch (e) {
      setError(
        controller.signal.aborted
          ? "Queueing stopped. Requests already accepted remain queued."
          : `${message(e)} Queueing stopped; accepted requests remain queued. You can retry the selection safely.`,
      );
    } finally {
      bulkAbort.current = null;
      setBulkProgress(null);
      setBusy(false);
      await refresh();
    }
  }
  return (
    <Box sx={{ p: 2, maxWidth: 1500 }}>
      <Title title="Translations" />
      <Typography variant="h4" gutterBottom>
        Translations
      </Typography>
      <Typography gutterBottom>
        Review English source fields and Swedish drafts. Only Approve selected
        changes published content.
      </Typography>
      {overview.isPending && (
        <CircularProgress aria-label="Loading translations" />
      )}
      {overview.error && (
        <Alert severity="error">{message(overview.error)}</Alert>
      )}
      {overview.data && (
        <Alert severity="info">
          {overview.data.aiEnabled
            ? "AI generation is enabled."
            : "AI generation is disabled. Manual translation and review remain available."}{" "}
          Daily token usage: {overview.data.usage.used.toLocaleString("en-GB")}{" "}
          used + {overview.data.usage.reserved.toLocaleString("en-GB")} reserved
          / {overview.data.dailyBudget.toLocaleString("en-GB")} (
          {overview.data.usage.day}, Europe/Stockholm).
        </Alert>
      )}
      <Paper component="fieldset" variant="outlined" sx={{ p: 2, my: 2 }}>
        <Typography component="legend">Select entire resources</Typography>
        <Typography variant="body2">
          Includes all records across all pages, independent of the search and
          status filters below. Only eligible missing or outdated translations
          generate suggestions. Manual translations and local edits are
          preserved. Unchanged drafts and rejected suggestions require explicit
          regeneration.
        </Typography>
        <FormControlLabel
          label="Select all eligible resources"
          control={
            <Checkbox
              disabled={busy || !eligibleResources.length}
              checked={
                eligibleResources.length > 0 &&
                bulkResources.length === eligibleResources.length
              }
              indeterminate={
                bulkResources.length > 0 &&
                bulkResources.length < eligibleResources.length
              }
              onChange={(e) =>
                setSelectedResources(
                  e.target.checked ? eligibleResources.map((r) => r.id) : [],
                )
              }
            />
          }
        />
        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
          {resources.map((r) => (
            <FormControlLabel
              key={r.id}
              label={`Entire resource: ${r.label}`}
              control={
                <Checkbox
                  checked={selectedResources.includes(r.id)}
                  disabled={busy || !r.canEdit || !r.aiAllowed}
                  onChange={(e) =>
                    setSelectedResources((old) =>
                      e.target.checked
                        ? [...old, r.id]
                        : old.filter((id) => id !== r.id),
                    )
                  }
                />
              }
            />
          ))}
        </Box>
        <Button
          disabled={busy || !overview.data?.aiEnabled || !bulkResources.length}
          onClick={generateBulk}
        >
          Generate for selected resources ({bulkResources.length})
        </Button>
        {bulkProgress && (
          <>
            <Typography role="status">
              Queueing {bulkProgress.resource}: {bulkProgress.checked} records
              checked; {bulkProgress.jobs} jobs queued or already scheduled.
            </Typography>
            <Button onClick={() => bulkAbort.current?.abort()}>
              Stop queueing
            </Button>
          </>
        )}
        {bulkResult && <Alert severity="info">{bulkResult}</Alert>}
      </Paper>
      <Stack direction={{ xs: "column", md: "row" }} spacing={2} sx={{ my: 2 }}>
        <TextField
          select
          id="translation-resource"
          label="Resource"
          value={resource?.id ?? ""}
          onChange={(e) => {
            setChoice(e.target.value);
            setPage(0);
            setSelected([]);
          }}
          sx={{ minWidth: 250 }}
        >
          {resources.map((r) => (
            <MenuItem key={r.id} value={r.id}>
              {r.label} ({r.missingRecords})
            </MenuItem>
          ))}
        </TextField>
        <TextField
          label="Search translations"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(0);
            setSelected([]);
          }}
        />
        <TextField
          select
          id="translation-status"
          label="Status"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(0);
            setSelected([]);
          }}
        >
          {statuses.map(([value, label]) => (
            <MenuItem key={value} value={value}>
              {label}
            </MenuItem>
          ))}
        </TextField>
        <Button
          onClick={() => generate(selected)}
          disabled={
            busy ||
            !resource?.canEdit ||
            !resource.aiAllowed ||
            !overview.data?.aiEnabled ||
            !selected.length
          }
        >
          Generate selected ({selected.length}/50)
        </Button>
      </Stack>
      {resource && !resource.aiAllowed && (
        <Alert severity="info">
          Exhibitor descriptions belong to exhibitors. They are never sent to
          AI; manual translation is available.
        </Alert>
      )}
      {error && (
        <Alert severity="error" onClose={() => setError("")}>
          {error}
        </Alert>
      )}
      {records.error && (
        <Alert severity="error">{message(records.error)}</Alert>
      )}
      {records.isPending && resource && (
        <CircularProgress aria-label="Loading records" />
      )}
      {resource &&
        records.data?.records.map((record) => (
          <TranslationRecordCard
            key={keyFor(resource.id, record.id)}
            record={record}
            resource={resource}
            busy={busy}
            selected={selected.includes(record.id)}
            onSelect={(checked) =>
              setSelected((old) =>
                checked
                  ? [...old, record.id].slice(0, 50)
                  : old.filter((id) => id !== record.id),
              )
            }
            generate={(regenerate) => generate([record.id], regenerate)}
            aiEnabled={overview.data?.aiEnabled ?? false}
            onError={setError}
            refresh={refresh}
          />
        ))}
      {records.data?.records.length === 0 && (
        <Typography sx={{ my: 3 }}>No matching translation records.</Typography>
      )}
      <TablePagination
        component="div"
        count={records.data?.total ?? 0}
        page={page}
        rowsPerPage={10}
        rowsPerPageOptions={[10]}
        onPageChange={(_, next) => {
          setPage(next);
        }}
      />
    </Box>
  );
}
function TranslationRecordCard({
  record,
  resource,
  busy,
  selected,
  onSelect,
  generate,
  aiEnabled,
  onError,
  refresh,
}: {
  record: TranslationRecord;
  resource: Resource;
  busy: boolean;
  selected: boolean;
  onSelect: (selected: boolean) => void;
  generate: (regenerate?: boolean) => Promise<void>;
  aiEnabled: boolean;
  onError: (message: string) => void;
  refresh: () => Promise<unknown>;
}) {
  const { drafts, setDrafts } = useTranslationDrafts();
  const basename = useBasename();
  const key = keyFor(resource.id, record.id);
  const draft = drafts[key];
  const base = draft?.record ?? record;
  const selectedFields = draft?.selected ?? [];
  const [saving, setSaving] = useState(false);
  function change(update: (value: TranslationDraft) => TranslationDraft) {
    setDrafts((old) => ({
      ...old,
      [key]: update(old[key] ?? { record, texts: {}, selected: [] }),
    }));
  }
  async function review(action: "draft" | "approve" | "reject") {
    if (!selectedFields.length) return;
    setSaving(true);
    onError("");
    try {
      const updated = await translationRequest<TranslationRecord>(
        `/${resource.id}/${record.id}/review`,
        {
          action,
          fingerprint: base.fingerprint,
          fields: selectedFields.map((id) => {
            const field = base.fields.find((f) => f.id === id)!;
            const suggestion =
              field.suggestion?.status === "draft"
                ? field.suggestion
                : undefined;
            return {
              fieldId: id,
              suggestionId: suggestion?.id ?? 0,
              version: suggestion?.version ?? 0,
              text: draft?.texts[id] ?? fieldValue(field),
              targetBefore: field.published,
            };
          }),
        },
      );
      setDrafts((old) => {
        const next = { ...old };
        const current = next[key];
        if (current) {
          const texts = { ...current.texts };
          for (const id of selectedFields) delete texts[id];
          if (Object.keys(texts).length) {
            next[key] = {
              record: updated,
              texts,
              selected: current.selected.filter(
                (id) => !selectedFields.includes(id),
              ),
            };
          } else delete next[key];
        }
        return next;
      });
      await refresh();
    } catch (e) {
      onError(message(e));
    } finally {
      setSaving(false);
    }
  }
  async function cancel() {
    setSaving(true);
    try {
      await translationRequest(
        `/${resource.id}/${record.id}/jobs`,
        undefined,
        "DELETE",
      );
      await refresh();
    } catch (e) {
      onError(message(e));
    } finally {
      setSaving(false);
    }
  }
  return (
    <Paper variant="outlined" sx={{ p: 2, my: 2 }}>
      <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
        <FormControlLabel
          control={
            <Checkbox
              checked={selected}
              disabled={!resource.canEdit || !resource.aiAllowed}
              onChange={(e) => onSelect(e.target.checked)}
            />
          }
          label={`Select record ${record.id}`}
        />
        <Typography variant="h6">
          {record.label || `Record ${record.id}`}
        </Typography>
        <Button component={Link} to={`${basename}/${resource.id}/${record.id}`}>
          Open record
        </Button>
        {record.job && (
          <Chip
            label={record.job.status.replace(/_/g, " ")}
            color={record.job.status === "failed" ? "error" : "default"}
          />
        )}
      </Stack>
      {draft && base.fingerprint !== record.fingerprint && (
        <Alert severity="warning">
          The original changed. Your edits are preserved; reload the record and
          reconcile them before approval.
        </Alert>
      )}
      {record.job?.outdated && (
        <Alert severity="warning">
          The queued source has changed. This job will not generate suggestions
          from outdated content.
        </Alert>
      )}
      {record.job?.error && (
        <Alert severity="warning">
          Generation: {record.job.error.replace(/_/g, " ")}
        </Alert>
      )}
      {record.job?.status === "budget_wait" && (
        <Typography>
          Next attempt:{" "}
          {new Date(record.job.notBefore).toLocaleString("en-GB", {
            timeZone: "Europe/Stockholm",
          })}
        </Typography>
      )}
      <Box
        component="fieldset"
        disabled={saving}
        sx={{ border: 0, m: 0, p: 0, minWidth: 0 }}
      >
        {base.fields.map((field) => (
          <Box
            key={field.id}
            sx={{ borderTop: 1, borderColor: "divider", mt: 2, pt: 2 }}
          >
            <FormControlLabel
              control={
                <Checkbox
                  checked={selectedFields.includes(field.id)}
                  disabled={
                    !resource.canEdit ||
                    (!field.source.trim() &&
                      !field.published?.trim() &&
                      !field.suggestion)
                  }
                  onChange={(e) =>
                    change((current) => ({
                      ...current,
                      selected: e.target.checked
                        ? [...current.selected, field.id]
                        : current.selected.filter((id) => id !== field.id),
                    }))
                  }
                />
              }
              label={`${formatLabel(field.id)} (${field.status.replace(/_/g, " ")})`}
            />
            <Box
              sx={{
                display: "grid",
                gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
                gap: 2,
              }}
            >
              <Box>
                <Typography variant="subtitle2">
                  English original ({field.format})
                </Typography>
                <Typography
                  component="pre"
                  sx={{
                    whiteSpace: "pre-wrap",
                    overflowWrap: "anywhere",
                    fontFamily: "inherit",
                  }}
                >
                  {field.source || "No original text"}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">Swedish draft</Typography>
                {field.format === "markdown" && resource.canEdit ? (
                  <MarkdownEditor
                    label={`${formatLabel(field.id)} (Swedish)`}
                    value={draft?.texts[field.id] ?? fieldValue(field)}
                    onChange={(text) =>
                      change((current) => ({
                        ...current,
                        texts: { ...current.texts, [field.id]: text ?? "" },
                      }))
                    }
                    enableImageFeatures={false}
                  />
                ) : (
                  <TextField
                    fullWidth
                    multiline
                    minRows={2}
                    label={`${formatLabel(field.id)} (Swedish)`}
                    value={draft?.texts[field.id] ?? fieldValue(field)}
                    disabled={!resource.canEdit}
                    onChange={(e) =>
                      change((current) => ({
                        ...current,
                        texts: { ...current.texts, [field.id]: e.target.value },
                      }))
                    }
                  />
                )}
                <Typography
                  variant="caption"
                  component="p"
                  sx={{ mt: 1, whiteSpace: "pre-wrap" }}
                >
                  Published Swedish: {field.published ?? "Not translated"}
                </Typography>
              </Box>
            </Box>
            {field.status === "stale" && (
              <Alert severity="warning">
                This proposal uses an older original or published value. Reject
                it before starting a new manual draft, or request regeneration.
              </Alert>
            )}
            {field.publishedOutdated && (
              <Alert severity="warning">
                Published Swedish is based on an older English original. It
                stays published until a replacement is approved.
              </Alert>
            )}
            {field.warnings?.map((warning) => (
              <Alert severity="warning" key={warning}>
                {warning}
              </Alert>
            ))}
            {field.olderPolicy && (
              <Alert severity="warning">
                This suggestion uses older translation instructions or glossary.
                Review carefully or regenerate.
              </Alert>
            )}
            {field.suggestion && (
              <Typography variant="caption" component="p">
                {field.suggestion.model || "Manual draft"} ·{" "}
                {new Date(field.suggestion.createdAt).toLocaleString("en-GB", {
                  timeZone: "Europe/Stockholm",
                })}{" "}
                · Prompt {field.suggestion.promptVersion.slice(0, 12)} ·
                Glossary {field.suggestion.glossaryVersion.slice(0, 12)}
              </Typography>
            )}
          </Box>
        ))}
      </Box>
      <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ mt: 2 }}>
        {draft && (
          <Button
            disabled={saving}
            onClick={() => {
              if (
                !window.confirm(
                  "Discard local edits for this record and load its latest values?",
                )
              )
                return;
              setDrafts((old) => {
                const next = { ...old };
                delete next[key];
                return next;
              });
            }}
          >
            Discard local edits
          </Button>
        )}
        <Button
          disabled={
            busy ||
            saving ||
            !aiEnabled ||
            !resource.canEdit ||
            !resource.aiAllowed
          }
          onClick={() => generate(false)}
        >
          Generate suggestions
        </Button>
        <Button
          disabled={
            busy ||
            saving ||
            !aiEnabled ||
            !resource.canEdit ||
            !resource.aiAllowed
          }
          onClick={() => generate(true)}
        >
          Regenerate suggestions
        </Button>
        <Button
          disabled={saving || !resource.canEdit || !selectedFields.length}
          onClick={() => review("draft")}
        >
          Save draft
        </Button>
        <Button
          variant="contained"
          disabled={saving || !resource.canEdit || !selectedFields.length}
          onClick={() => review("approve")}
        >
          Approve selected
        </Button>
        <Button
          disabled={saving || !resource.canEdit || !selectedFields.length}
          onClick={() => review("reject")}
        >
          Reject selected
        </Button>
        {record.job &&
          ["queued", "budget_wait"].includes(record.job.status) && (
            <Button disabled={saving || !resource.canEdit} onClick={cancel}>
              Cancel generation
            </Button>
          )}
      </Stack>
    </Paper>
  );
}
