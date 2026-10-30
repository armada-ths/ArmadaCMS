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
  const [resource, setResource] = useState<string>(
    available[0]?.[0] ?? "exhibitors",
  );
  const [saving, setSaving] = useState(false);
  const notify = useNotify();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["eventro-review", resource, syncVersion],
    enabled: open && syncVersion === null && available.length > 0,
    queryFn: async () =>
      (
        await axiosInstance.get<ReviewRow[]>("/eventrosyncreview", {
          params: { resource },
        })
      ).data,
  });

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

  if (available.length === 0) return null;
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
          <Alert severity="info" sx={{ mb: 2 }}>
            CMS corrections are preserved. Keep CMS acknowledges a difference;
            use Eventro applies its current value and resumes synchronization
            for that field.
          </Alert>
          <TextField
            select
            label="Resource"
            value={resource}
            onChange={(event) => setResource(event.target.value)}
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
                <Stack direction="row" sx={{ flexWrap: "wrap", gap: 1, mb: 1 }}>
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
                    sx={{ borderTop: 1, borderColor: "divider", pt: 2, mt: 2 }}
                  >
                    <Typography fontWeight="bold">
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
