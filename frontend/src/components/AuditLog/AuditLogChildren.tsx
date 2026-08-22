import { Alert, Box, CircularProgress, Stack, Typography } from "@mui/material";
import {
  DateField,
  FunctionField,
  RecordContextProvider,
  ShowButton,
  TextField,
  useGetList,
  useRecordContext,
} from "react-admin";

type AuditLogRecord = {
  id: number;
  child_count?: number;
};

export const AuditLogChildren = () => {
  const parent = useRecordContext<AuditLogRecord>();
  const { data, error, isPending } = useGetList(
    "auditlogs",
    {
      filter: { parent_id: String(parent?.id) },
      pagination: { page: 1, perPage: Math.max(parent?.child_count ?? 0, 100) },
      sort: { field: "created_at", order: "ASC" },
    },
    { enabled: Boolean(parent?.id) },
  );

  if (!parent) return null;
  if (!parent.child_count) return null;
  if (isPending) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 2 }}>
        <CircularProgress
          size={24}
          aria-label="Loading related audit entries"
        />
      </Box>
    );
  }
  if (error) {
    return (
      <Alert severity="error">Could not load related audit entries.</Alert>
    );
  }
  if (!data?.length) {
    return <Alert severity="info">No related audit entries.</Alert>;
  }

  return (
    <Stack spacing={1} sx={{ p: 2 }}>
      {data.map((entry) => (
        <RecordContextProvider key={entry.id} value={entry}>
          <Box
            sx={{
              alignItems: "center",
              display: "grid",
              gap: 1,
              gridTemplateColumns:
                "minmax(140px, 1fr) repeat(3, minmax(100px, 1fr)) auto",
              p: 1,
            }}
          >
            <DateField source="created_at" showTime />
            <TextField source="action" />
            <FunctionField
              render={(record) => (
                <Typography variant="body2">
                  {String(record.resource_type)} #{String(record.resource_id)}
                </Typography>
              )}
            />
            <TextField source="actor_username" emptyText="System" />
            <ShowButton resource="auditlogs" />
          </Box>
        </RecordContextProvider>
      ))}
    </Stack>
  );
};
