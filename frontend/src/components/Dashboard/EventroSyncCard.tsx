import { useState } from "react";
import { useNotify } from "react-admin";
import {
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  Stack,
  Typography,
} from "@mui/material";
import { Sync } from "@mui/icons-material";
import globalApi from "../../context/globalApi";
import { hasPerm } from "../../utils/permissions";

const EVENTRO_ENDPOINTS = [
  { name: "Exhibitors", path: "/eventroexhibitors" },
  { name: "Events", path: "/eventroevents" },
  { name: "Members", path: "/eventromembers" },
  { name: "Recruitments", path: "/eventrorecruitments" },
] as const;

export const EventroSyncCard = ({ perms }: { perms: string[] }) => {
  const notify = useNotify();
  const [syncing, setSyncing] = useState<string | null>(null);

  if (!hasPerm(perms, "eventrosync.access")) return null;

  const handleSync = async (name: string, path: string) => {
    setSyncing(name);
    try {
      const res = await fetch(`${globalApi()}${path}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("accessToken")}`,
        },
      });
      const message = await res.text();
      if (!res.ok) {
        notify(`${name}: ${res.status} ${message}`, { type: "error" });
      } else {
        notify(`${name}: ${message}`, { type: "success" });
      }
    } catch (err) {
      notify(`${name}: Unexpected error: ${err}`, { type: "error" });
    } finally {
      setSyncing(null);
    }
  };

  return (
    <Card variant="outlined" sx={{ mt: 2 }}>
      <CardContent>
        <Stack direction="row" alignItems="center" spacing={1} mb={1.5}>
          <Sync color="action" />
          <Typography variant="h6">Eventro sync</Typography>
        </Stack>
        <Divider sx={{ mb: 2 }} />
        <Stack direction="row" flexWrap="wrap" gap={1.5}>
          {EVENTRO_ENDPOINTS.map(({ name, path }) => (
            <Button
              key={path}
              variant="outlined"
              size="small"
              disabled={syncing !== null}
              startIcon={
                syncing === name ? <CircularProgress size={14} /> : <Sync />
              }
              onClick={() => handleSync(name, path)}
            >
              Sync {name}
            </Button>
          ))}
        </Stack>
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: "block", mt: 1.5 }}
        >
          Recruitment and profile sync are non-destructive and only fill missing
          data.
        </Typography>
      </CardContent>
    </Card>
  );
};
