import { useGetList } from "react-admin";
import { useNavigate } from "react-router";
import {
  Box,
  Card,
  CardContent,
  Chip,
  Divider,
  Skeleton,
  Stack,
  Typography,
} from "@mui/material";
import { Cancel, CheckCircle, ToggleOn } from "@mui/icons-material";
import { hasPerm } from "../../utils/permissions";

interface FeatureFlag {
  id: number;
  key: string;
  description: string;
  enabled: boolean;
}

const prettifyFlagKey = (key: string): string =>
  key
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(" ");

export const FeatureFlagsSummary = ({ perms }: { perms: string[] }) => {
  const navigate = useNavigate();
  const { data, isPending } = useGetList<FeatureFlag>(
    "featureflags",
    {
      pagination: { page: 1, perPage: 100 },
      sort: { field: "key", order: "ASC" },
    },
    { enabled: hasPerm(perms, "featureflags.list") },
  );

  if (!hasPerm(perms, "featureflags.list")) return null;

  return (
    <Card variant="outlined" sx={{ mt: 2 }}>
      <CardContent>
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          mb={1}
        >
          <Stack direction="row" alignItems="center" spacing={1}>
            <ToggleOn color="action" />
            <Typography variant="h6">Feature flags</Typography>
          </Stack>
          <Typography
            variant="body2"
            color="primary"
            sx={{
              cursor: "pointer",
              "&:hover": { textDecoration: "underline" },
            }}
            onClick={() => navigate("/admin/featureflags")}
          >
            Manage →
          </Typography>
        </Stack>
        <Divider sx={{ mb: 1.5 }} />
        {isPending ? (
          <Skeleton height={36} />
        ) : (
          <Box display="flex" flexWrap="wrap" gap={1}>
            {(data ?? []).map((flag) => (
              <Chip
                key={flag.id}
                label={prettifyFlagKey(flag.key)}
                size="small"
                color={flag.enabled ? "success" : "default"}
                variant={flag.enabled ? "filled" : "outlined"}
                icon={flag.enabled ? <CheckCircle /> : <Cancel />}
              />
            ))}
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
