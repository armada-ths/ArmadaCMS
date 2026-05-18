import { useGetList } from "react-admin";
import { useNavigate } from "react-router";
import {
  Box,
  Card,
  CardContent,
  Divider,
  List,
  ListItem,
  ListItemText,
  Skeleton,
  Stack,
  Typography,
} from "@mui/material";
import { History } from "@mui/icons-material";
import { hasPerm } from "../../utils/permissions";

const ACTION_LABELS: Record<string, string> = {
  create: "created",
  update: "updated",
  delete: "deleted",
};

interface AuditEntry {
  id: number;
  action: string;
  resource_type: string;
  actor_username: string;
  created_at: string;
}

export const RecentActivity = ({ perms }: { perms: string[] }) => {
  const navigate = useNavigate();
  const { data, isPending } = useGetList<AuditEntry>(
    "auditlogs",
    {
      sort: { field: "created_at", order: "DESC" },
      pagination: { page: 1, perPage: 5 },
    },
    { enabled: hasPerm(perms, "auditlogs.list") },
  );

  if (!hasPerm(perms, "auditlogs.list")) return null;

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          mb={1}
        >
          <Stack direction="row" alignItems="center" spacing={1}>
            <History color="action" />
            <Typography variant="h6">Recent activity</Typography>
          </Stack>
          <Typography
            variant="body2"
            color="primary"
            sx={{
              cursor: "pointer",
              "&:hover": { textDecoration: "underline" },
            }}
            onClick={() => navigate("/admin/auditlogs")}
          >
            View all →
          </Typography>
        </Stack>
        <Divider />
        {isPending ? (
          <Stack spacing={1} mt={1}>
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} height={40} />
            ))}
          </Stack>
        ) : (
          <List dense disablePadding>
            {(data ?? []).map((entry, idx) => (
              <Box key={entry.id}>
                <ListItem disableGutters>
                  <ListItemText
                    primary={
                      <Typography variant="body2">
                        <strong>{entry.actor_username}</strong>{" "}
                        {ACTION_LABELS[entry.action] ?? entry.action}{" "}
                        <em>{entry.resource_type}</em>
                      </Typography>
                    }
                    secondary={new Date(entry.created_at).toLocaleString()}
                  />
                </ListItem>
                {idx < (data?.length ?? 0) - 1 && <Divider component="li" />}
              </Box>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  );
};
