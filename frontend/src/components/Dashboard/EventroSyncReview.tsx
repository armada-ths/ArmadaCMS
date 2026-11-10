import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNotify } from "react-admin";
import { Link as RouterLink } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from "@mui/material";
import axiosInstance from "../../context/axiosInstance";
import { hasPerm } from "../../utils/permissions";

const resources = [
  ["exhibitors", "Exhibitors"],
  ["events", "Events"],
  ["profiles", "Member profiles"],
  ["fairdates", "Fair dates"],
  ["recruitmentperiods", "Recruitment periods"],
  ["recruitmentroles", "Recruitment roles"],
] as const;

const blockableResources = [
  ...resources,
  ["industries", "Industries"],
  ["employments", "Employment types"],
  ["programs", "Fields of study"],
] as const;

const fieldLabels: Record<string, string> = {
  name: "Name",
  tier: "Package",
  companyWebsite: "Website",
  about: "Company description",
  logo: "Logo",
  cities: "Locations",
  industries: "Industries",
  employments: "Employment opportunities",
  programs: "Education",
  description: "Description",
  rank: "Member group",
  title: "Role",
  email: "Email",
  photo: "Profile photo",
  eventStart: "Event starts",
  eventEnd: "Event ends",
  registrationEnd: "Registration closes",
  eventMaxCapacity: "Capacity",
  location: "Location",
  food: "Food",
  openTo: "Open to",
  mapUrl: "Map link",
  signupLink: "Registration link",
  fairDays: "Fair days",
  irStart: "Initial registration opens",
  irEnd: "Initial registration closes",
  irAcceptance: "Initial registration acceptance",
  frStart: "Final registration opens",
  frEnd: "Final registration closes",
  eventsStart: "Events start",
  link: "Link",
  startDate: "Opens",
  endDate: "Closes",
  team_id: "Team ID",
  recruitmentId: "Recruitment period ID",
};

type ReviewField = {
  field: string;
  cms: unknown;
  eventro: unknown;
  conflict: boolean;
};
type ReviewRow = {
  resource: string;
  id: number;
  name: string;
  missing: string[];
  fields: ReviewField[];
  notice?: string;
  unmappedProducts:
    { productId: string; name: string; displayName: string }[] | null;
};

type BlockedRecord = {
  id: number;
  eventroKey: string;
  label: string;
  deletedAt: string;
};

type ReviewTab = "review" | "blocks";

function displayValue(value: unknown) {
  if (value === null || value === "") return "(empty)";
  return typeof value === "string" ? value : JSON.stringify(value, null, 2);
}

export function EventroSyncReview({
  perms,
  syncVersion,
}: {
  perms: string[];
  syncVersion: string | null;
}) {
  const [open, setOpen] = useState(false);
  const available = resources.filter(([key]) => hasPerm(perms, `${key}.edit`));
  const blockResources = blockableResources.filter(([key]) =>
    hasPerm(perms, key + ".delete"),
  );
  const [reviewResource, setReviewResource] = useState<string>("");
  const [blockResource, setBlockResource] = useState<string>("");
  const [selectedTab, setSelectedTab] = useState<ReviewTab>("review");
  const tab =
    available.length === 0
      ? "blocks"
      : blockResources.length === 0
        ? "review"
        : selectedTab;
  const resource = available.some(([key]) => key === reviewResource)
    ? reviewResource
    : (available[0]?.[0] ?? "");
  const blockedResource = blockResources.some(([key]) => key === blockResource)
    ? blockResource
    : (blockResources[0]?.[0] ?? "");
  const [saving, setSaving] = useState(false);
  const notify = useNotify();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["eventro-review", resource, syncVersion],
    enabled: open && tab === "review" && syncVersion === null && !!resource,
    queryFn: async () =>
      (
        await axiosInstance.get<ReviewRow[]>("/eventrosyncreview", {
          params: { resource },
        })
      ).data,
  });

  const blocks = useQuery({
    queryKey: ["eventro-blocks", blockedResource],
    enabled: open && tab === "blocks" && !!blockedResource,
    queryFn: async () =>
      (
        await axiosInstance.get<BlockedRecord[]>("/eventrosyncblocks", {
          params: { resource: blockedResource },
        })
      ).data,
  });

  const allowReimport = async (record: BlockedRecord) => {
    setSaving(true);
    try {
      await axiosInstance.delete("/eventrosyncblocks/" + record.id);
      await queryClient.invalidateQueries({ queryKey: ["eventro-blocks"] });
      notify("Eventro can import this record on the next sync.", {
        type: "success",
      });
    } catch {
      await queryClient.invalidateQueries({ queryKey: ["eventro-blocks"] });
      notify(
        "Could not allow reimport. The blocked records list was refreshed.",
        {
          type: "error",
        },
      );
    } finally {
      setSaving(false);
    }
  };

  const resolve = async (
    row: ReviewRow,
    field: ReviewField,
    choice: "cms" | "eventro",
  ) => {
    setSaving(true);
    try {
      await axiosInstance.post("/eventrosyncreview", {
        resource: row.resource,
        recordId: row.id,
        field: field.field,
        choice,
        cms: field.cms,
        eventro: field.eventro,
      });
      await queryClient.invalidateQueries({ queryKey: ["eventro-review"] });
      await queryClient.invalidateQueries({ queryKey: [row.resource] });
      notify(
        choice === "cms"
          ? "CMS correction retained."
          : "Eventro value applied; future changes will sync.",
        { type: "success" },
      );
    } catch {
      notify(
        "Could not save the decision. Refresh the review; values or permissions may have changed.",
        { type: "error" },
      );
    } finally {
      setSaving(false);
    }
  };

  if (available.length === 0 && blockResources.length === 0) return null;
  return (
    <>
      <Button
        sx={{ mt: 2 }}
        variant="outlined"
        onClick={() => setOpen(true)}
        disabled={syncVersion !== null}
      >
        Review sync and data quality
      </Button>
      <Dialog
        open={open}
        onClose={() => !saving && setOpen(false)}
        fullWidth
        maxWidth="lg"
      >
        <DialogTitle>Eventro sync and data quality</DialogTitle>
        <DialogContent>
          <Tabs
            value={tab}
            onChange={(_, value: ReviewTab) => setSelectedTab(value)}
            aria-label="Eventro review views"
            sx={{ mb: 2 }}
          >
            {available.length > 0 && (
              <Tab
                id="eventro-review-tab"
                aria-controls="eventro-review-panel"
                value="review"
                label="Field review"
                disabled={saving}
              />
            )}
            {blockResources.length > 0 && (
              <Tab
                id="eventro-blocks-tab"
                aria-controls="eventro-blocks-panel"
                value="blocks"
                label="Blocked records"
                disabled={saving}
              />
            )}
          </Tabs>
          {tab === "blocks" && (
            <Stack
              id="eventro-blocks-panel"
              role="tabpanel"
              aria-labelledby="eventro-blocks-tab"
              spacing={2}
            >
              <Alert severity="info">
                Eventro skips these records. Allow reimport to let the next sync
                create a fresh record from Eventro. Recruitment roles are
                unblocked separately from their period.
              </Alert>
              <TextField
                select
                label="Resource"
                value={blockedResource}
                onChange={(event) => setBlockResource(event.target.value)}
                disabled={saving}
                fullWidth
              >
                {blockResources.map(([key, label]) => (
                  <MenuItem key={key} value={key}>
                    {label}
                  </MenuItem>
                ))}
              </TextField>
              {blocks.isFetching && (
                <CircularProgress
                  size={20}
                  aria-label="Loading blocked records"
                />
              )}
              {blocks.isError && (
                <Alert
                  severity="error"
                  action={
                    <Button
                      color="inherit"
                      onClick={() => void blocks.refetch()}
                    >
                      Retry
                    </Button>
                  }
                >
                  Could not load blocked records.
                </Alert>
              )}
              {blocks.isSuccess && blocks.data.length === 0 && (
                <Alert severity="success">No blocked records.</Alert>
              )}
              {(blocks.data ?? []).map((record) => (
                <Stack
                  key={record.id}
                  direction={{ xs: "column", sm: "row" }}
                  spacing={1}
                  sx={{
                    justifyContent: "space-between",
                    alignItems: { sm: "center" },
                  }}
                >
                  <Box sx={{ overflowWrap: "anywhere" }}>
                    <Typography sx={{ fontWeight: "bold" }}>
                      {record.label || record.eventroKey}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {record.eventroKey} ·{" "}
                      {new Date(record.deletedAt).toLocaleString()}
                    </Typography>
                  </Box>
                  <Button
                    disabled={saving}
                    onClick={() => void allowReimport(record)}
                  >
                    Allow reimport
                  </Button>
                </Stack>
              ))}
            </Stack>
          )}
          {tab === "review" && (
            <Box
              id="eventro-review-panel"
              role="tabpanel"
              aria-labelledby="eventro-review-tab"
            >
              <Alert severity="info" sx={{ mb: 2 }}>
                CMS corrections are preserved. Keep CMS acknowledges a
                difference; use Eventro applies its current value and resumes
                synchronization for that field.
              </Alert>
              <TextField
                select
                label="Resource"
                value={resource}
                onChange={(event) => setReviewResource(event.target.value)}
                disabled={saving}
                fullWidth
                sx={{ mb: 2 }}
              >
                {available.map(([key, label]) => (
                  <MenuItem key={key} value={key}>
                    {label}
                  </MenuItem>
                ))}
              </TextField>
              <Button
                onClick={() => void query.refetch()}
                disabled={saving || query.isFetching}
              >
                Refresh review
              </Button>
              {query.isFetching && (
                <CircularProgress size={20} aria-label="Loading review" />
              )}
              {query.isError && (
                <Alert severity="error">Could not load the review.</Alert>
              )}
              {query.isSuccess && query.data.length === 0 && (
                <Alert severity="success">No items need review.</Alert>
              )}
              <Stack spacing={3} sx={{ mt: 2 }}>
                {(query.data ?? []).map((row) => (
                  <Box
                    key={row.id}
                    sx={{
                      border: 1,
                      borderColor: "divider",
                      borderRadius: 1,
                      p: 2,
                    }}
                  >
                    <Button
                      component={RouterLink}
                      to={`/${row.resource}/${row.id}`}
                      disabled={saving}
                    >
                      {row.name || `Record ${row.id}`}
                    </Button>
                    {row.notice && (
                      <Alert severity="warning" sx={{ mb: 1 }}>
                        {row.notice}
                      </Alert>
                    )}
                    <Stack
                      direction="row"
                      sx={{ flexWrap: "wrap", gap: 1, mb: 1 }}
                    >
                      {row.missing.map((field) => (
                        <Chip
                          key={field}
                          label={`Missing: ${fieldLabels[field] ?? field}`}
                          size="small"
                          color="warning"
                        />
                      ))}
                    </Stack>
                    {!!row.unmappedProducts?.length && (
                      <Alert severity="warning" sx={{ mb: 2 }}>
                        Products without a tier mapping (including possible
                        add-ons):
                        {row.unmappedProducts.map((product) => (
                          <Typography key={product.productId} variant="body2">
                            {product.displayName || product.name} —{" "}
                            {product.productId}
                          </Typography>
                        ))}
                      </Alert>
                    )}
                    {row.fields.map((field) => (
                      <Box
                        key={field.field}
                        sx={{
                          borderTop: 1,
                          borderColor: "divider",
                          pt: 2,
                          mt: 2,
                        }}
                      >
                        <Typography sx={{ fontWeight: "bold" }}>
                          {fieldLabels[field.field] ?? field.field} —{" "}
                          {field.conflict ? "Review needed" : "CMS override"}
                        </Typography>
                        <Box
                          sx={{
                            display: "grid",
                            gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
                            gap: 2,
                            my: 1,
                          }}
                        >
                          {(
                            [
                              ["CMS", field.cms],
                              ["Eventro", field.eventro],
                            ] as const
                          ).map(([label, value]) => (
                            <Box key={label}>
                              <Typography variant="caption">{label}</Typography>
                              <Box
                                component="pre"
                                sx={{
                                  whiteSpace: "pre-wrap",
                                  overflowWrap: "anywhere",
                                  m: 0,
                                  fontFamily: "inherit",
                                }}
                              >
                                {displayValue(value)}
                              </Box>
                            </Box>
                          ))}
                        </Box>
                        <Stack direction="row" spacing={1}>
                          <Button
                            disabled={saving}
                            onClick={() => void resolve(row, field, "cms")}
                          >
                            Keep CMS
                          </Button>
                          <Button
                            disabled={saving}
                            variant="outlined"
                            onClick={() => void resolve(row, field, "eventro")}
                          >
                            Use Eventro
                          </Button>
                        </Stack>
                      </Box>
                    ))}
                  </Box>
                ))}
              </Stack>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button disabled={saving} onClick={() => setOpen(false)}>
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
