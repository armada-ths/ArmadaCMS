import { useState } from "react";
import { useNotify } from "react-admin";
import {
  Alert,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import { Sync } from "@mui/icons-material";
import axios from "axios";
import axiosInstance from "../../context/axiosInstance";
import { hasPerm } from "../../utils/permissions";

const EVENTRO_ENDPOINTS = [
  { name: "Exhibitors", path: "/eventroexhibitors", requiresFair: true },
  { name: "Events", path: "/eventroevents", requiresFair: true },
  { name: "Fair dates", path: "/eventrofairdates", requiresFair: true },
  { name: "Members", path: "/eventromembers", requiresFair: false },
  {
    name: "Recruitments",
    path: "/eventrorecruitments",
    requiresFair: false,
  },
] as const;

type EventroEndpoint = (typeof EVENTRO_ENDPOINTS)[number];

type EventroFair = {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
};

type EventroFairsResponse = {
  fairInstances: EventroFair[];
};

const formatRequestError = (error: unknown) => {
  if (!axios.isAxiosError(error)) return `Unexpected error: ${error}`;

  const status = error.response?.status;
  const detail =
    typeof error.response?.data === "string"
      ? error.response.data.trim()
      : error.message;
  return [status, detail].filter(Boolean).join(" ");
};

export const EventroSyncCard = ({ perms }: { perms: string[] }) => {
  const notify = useNotify();
  const [syncing, setSyncing] = useState<string | null>(null);
  const [selectingFor, setSelectingFor] = useState<EventroEndpoint | null>(
    null,
  );
  const [fairInstances, setFairInstances] = useState<EventroFair[]>([]);
  const [selectedFairId, setSelectedFairId] = useState("");
  const [loadingFairs, setLoadingFairs] = useState(false);

  if (!hasPerm(perms, "eventrosync.access")) return null;

  const handleSync = async (name: string, path: string, fairId?: string) => {
    setSyncing(name);
    try {
      const response = await axiosInstance.get<string>(path, {
        params: fairId ? { fairId } : undefined,
        responseType: "text",
      });
      notify(`${name}: ${response.data}`, { type: "success" });
    } catch (err) {
      notify(`${name}: ${formatRequestError(err)}`, { type: "error" });
    } finally {
      setSyncing(null);
    }
  };

  const promptForFair = async (endpoint: EventroEndpoint) => {
    setSelectingFor(endpoint);
    setSelectedFairId("");
    setFairInstances([]);
    setLoadingFairs(true);

    try {
      const { data: result } =
        await axiosInstance.get<EventroFairsResponse>("/eventrofairs");
      setFairInstances(result.fairInstances);
      if (result.fairInstances.length === 0) {
        notify("Eventro returned no active fair instances.", {
          type: "warning",
        });
        setSelectingFor(null);
      }
    } catch (err) {
      notify(`Fair instances: ${formatRequestError(err)}`, { type: "error" });
      setSelectingFor(null);
    } finally {
      setLoadingFairs(false);
    }
  };

  const handleEndpointClick = (endpoint: EventroEndpoint) => {
    if (endpoint.requiresFair) {
      void promptForFair(endpoint);
      return;
    }
    void handleSync(endpoint.name, endpoint.path);
  };

  const handleSelectedFairSync = () => {
    if (!selectingFor || !selectedFairId) return;

    const endpoint = selectingFor;
    const fairId = selectedFairId;
    setSelectingFor(null);
    void handleSync(endpoint.name, endpoint.path, fairId);
  };

  return (
    <>
      <Card variant="outlined" sx={{ mt: 2 }}>
        <CardContent>
          <Stack direction="row" alignItems="center" spacing={1} mb={1.5}>
            <Sync color="action" />
            <Typography variant="h6">Eventro sync</Typography>
          </Stack>
          <Divider sx={{ mb: 2 }} />
          <Stack direction="row" flexWrap="wrap" gap={1.5}>
            {EVENTRO_ENDPOINTS.map((endpoint) => (
              <Button
                key={endpoint.path}
                variant="outlined"
                size="small"
                disabled={syncing !== null || loadingFairs}
                startIcon={
                  syncing === endpoint.name ? (
                    <CircularProgress size={14} />
                  ) : (
                    <Sync />
                  )
                }
                onClick={() => handleEndpointClick(endpoint)}
              >
                Sync {endpoint.name}
              </Button>
            ))}
          </Stack>
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ display: "block", mt: 1.5 }}
          >
            Recruitment and profile sync are non-destructive and only fill
            missing data.
          </Typography>
        </CardContent>
      </Card>

      <Dialog
        open={selectingFor !== null}
        onClose={() => !loadingFairs && setSelectingFor(null)}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>Select an Eventro fair</DialogTitle>
        <DialogContent>
          {selectingFor?.path === "/eventrofairdates" && !loadingFairs && (
            <Alert severity="warning" sx={{ mt: 1, mb: 2 }}>
              This will permanently delete all existing fair date entries and
              replace them with the selected Eventro timeline.
            </Alert>
          )}
          {loadingFairs ? (
            <Stack alignItems="center" sx={{ py: 3 }}>
              <CircularProgress aria-label="Loading fair instances" />
            </Stack>
          ) : (
            <FormControl fullWidth sx={{ mt: 1 }}>
              <InputLabel id="eventro-fair-label">Fair instance</InputLabel>
              <Select
                labelId="eventro-fair-label"
                value={selectedFairId}
                label="Fair instance"
                onChange={(event) => setSelectedFairId(event.target.value)}
              >
                {fairInstances.map((fair) => (
                  <MenuItem key={fair.id} value={fair.id}>
                    {fair.name} ({fair.startDate.slice(0, 10)} -{" "}
                    {fair.endDate.slice(0, 10)})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSelectingFor(null)} disabled={loadingFairs}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSelectedFairSync}
            disabled={loadingFairs || !selectedFairId}
          >
            Sync {selectingFor?.name}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
