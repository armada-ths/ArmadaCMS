import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Stack,
  Typography,
} from "@mui/material";
import { alpha, useTheme } from "@mui/material/styles";
import {
  DateField,
  Show,
  SimpleShowLayout,
  TextField,
  useRecordContext,
} from "react-admin";
import { AuditLogChildren } from "./AuditLogChildren";

type JsonPrimitive = string | number | boolean | null;
type JsonValue =
  | JsonPrimitive
  | JsonValue[]
  | {
      [key: string]: JsonValue;
    };

type DiffEntry = {
  path: string;
  oldValue: JsonValue | undefined;
  newValue: JsonValue | undefined;
};

type AuditLogRecord = Record<string, unknown>;

const parseJsonValue = (value: unknown): JsonValue => {
  if (value === null || value === undefined || value === "") {
    return null;
  }

  if (typeof value === "string") {
    try {
      return JSON.parse(value) as JsonValue;
    } catch {
      return value;
    }
  }

  return value as JsonValue;
};

const formatJsonValue = (value: unknown) =>
  JSON.stringify(parseJsonValue(value), null, 2) ?? "null";

const isPlainObject = (value: JsonValue): value is Record<string, JsonValue> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const asObject = (
  value: JsonValue | undefined,
): Record<string, JsonValue> | null =>
  value !== undefined && isPlainObject(value) ? value : null;

const valuesEqual = (
  left: JsonValue | undefined,
  right: JsonValue | undefined,
) => JSON.stringify(left) === JSON.stringify(right);

const normalizeForDiff = (value: JsonValue): JsonValue => {
  if (Array.isArray(value)) {
    return value.map(normalizeForDiff);
  }

  if (!isPlainObject(value)) {
    return value;
  }

  const normalized: Record<string, JsonValue> = {};

  for (const [key, rawChild] of Object.entries(value)) {
    normalized[key] = normalizeForDiff(rawChild);
  }

  for (const [key, child] of Object.entries(normalized)) {
    const relationIdKey = `${key}_id`;

    if (
      isPlainObject(child) &&
      "id" in child &&
      relationIdKey in normalized &&
      valuesEqual(child.id, normalized[relationIdKey])
    ) {
      delete normalized[key];
    }
  }

  return normalized;
};

const collectDiffEntries = (
  oldValue: JsonValue | undefined,
  newValue: JsonValue | undefined,
  currentPath = "",
): DiffEntry[] => {
  if (valuesEqual(oldValue, newValue)) {
    return [];
  }

  const pathLabel = currentPath || "(root)";

  if (Array.isArray(oldValue) && Array.isArray(newValue)) {
    const maxLength = Math.max(oldValue.length, newValue.length);
    const entries: DiffEntry[] = [];

    for (let index = 0; index < maxLength; index += 1) {
      entries.push(
        ...collectDiffEntries(
          oldValue[index],
          newValue[index],
          `${currentPath}[${index}]`,
        ),
      );
    }

    return entries;
  }

  const oldObject = asObject(oldValue);
  const newObject = asObject(newValue);

  if (oldObject && newObject) {
    const keys = new Set([
      ...Object.keys(oldObject),
      ...Object.keys(newObject),
    ]);
    const entries: DiffEntry[] = [];

    for (const key of keys) {
      const childPath = currentPath ? `${currentPath}.${key}` : key;
      entries.push(
        ...collectDiffEntries(oldObject[key], newObject[key], childPath),
      );
    }

    return entries;
  }

  return [
    {
      path: pathLabel,
      oldValue,
      newValue,
    },
  ];
};

const formatInlineValue = (value: JsonValue | undefined) => {
  if (value === undefined) {
    return "—";
  }

  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return String(value);
  }

  return JSON.stringify(value, null, 2);
};

const isSmallPayload = (formatted: string) =>
  formatted.split("\n").length <= 18 && formatted.length <= 800;

const PayloadViewer = ({
  value,
  title,
  autoExpand = false,
}: {
  value: unknown;
  title: string;
  autoExpand?: boolean;
}) => {
  const theme = useTheme();
  const formattedValue = formatJsonValue(value);
  const expanded = autoExpand && isSmallPayload(formattedValue);

  return (
    <Accordion
      disableGutters
      defaultExpanded={expanded}
      sx={{
        backgroundColor: alpha(theme.palette.background.paper, 0.55),
        color: theme.palette.text.primary,
        border: `1px solid ${alpha(theme.palette.divider, 0.9)}`,
        borderRadius: "6px",
        overflow: "hidden",
        boxShadow: "none",
        "&::before": { display: "none" },
      }}
    >
      <AccordionSummary
        expandIcon={<Box component="span">▾</Box>}
        sx={{
          minHeight: "unset",
          backgroundColor: alpha(theme.palette.action.hover, 0.35),
          "& .MuiAccordionSummary-content": {
            my: 1,
            alignItems: "center",
            gap: 1,
          },
        }}
      >
        <Typography variant="body2" color="text.secondary">
          {title}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {formattedValue.split("\n").length} lines
        </Typography>
      </AccordionSummary>
      <AccordionDetails sx={{ p: 0 }}>
        <Box
          component="pre"
          sx={{
            m: 0,
            p: 1.5,
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            fontFamily: "monospace",
            fontSize: "0.85rem",
            lineHeight: 1.5,
            backgroundColor:
              theme.palette.mode === "dark"
                ? alpha(theme.palette.common.white, 0.08)
                : alpha(theme.palette.common.black, 0.04),
            color: theme.palette.text.primary,
            maxHeight: "420px",
            overflow: "auto",
          }}
        >
          {formattedValue}
        </Box>
      </AccordionDetails>
    </Accordion>
  );
};

const ChangeSummary = ({ diffEntries }: { diffEntries: DiffEntry[] }) => {
  const theme = useTheme();
  const collapsedByDefault = diffEntries.length > 5;

  if (diffEntries.length === 0) {
    return (
      <Box
        sx={{
          p: 1.5,
          borderRadius: "6px",
          border: `1px solid ${alpha(theme.palette.divider, 0.9)}`,
          backgroundColor: alpha(theme.palette.success.main, 0.08),
          color: theme.palette.text.secondary,
        }}
      >
        No meaningful field-level changes detected.
      </Box>
    );
  }

  return (
    <Accordion
      disableGutters
      defaultExpanded={!collapsedByDefault}
      sx={{
        backgroundColor: alpha(theme.palette.background.paper, 0.55),
        color: theme.palette.text.primary,
        border: `1px solid ${alpha(theme.palette.divider, 0.9)}`,
        borderRadius: "6px",
        overflow: "hidden",
        boxShadow: "none",
        "&::before": { display: "none" },
      }}
    >
      <AccordionSummary
        expandIcon={<Box component="span">▾</Box>}
        sx={{
          minHeight: "unset",
          backgroundColor: alpha(theme.palette.warning.main, 0.1),
          "& .MuiAccordionSummary-content": {
            my: 1,
            alignItems: "center",
            gap: 1,
          },
        }}
      >
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {diffEntries.length} changed{" "}
          {diffEntries.length === 1 ? "field" : "fields"}
        </Typography>
      </AccordionSummary>
      <AccordionDetails sx={{ p: 1.5 }}>
        <Stack spacing={1.25}>
          {diffEntries.map((entry) => (
            <Box
              key={entry.path}
              sx={{
                p: 1.25,
                borderRadius: "6px",
                border: `1px solid ${alpha(theme.palette.divider, 0.8)}`,
                backgroundColor:
                  theme.palette.mode === "dark"
                    ? alpha(theme.palette.common.white, 0.04)
                    : alpha(theme.palette.common.black, 0.02),
              }}
            >
              <Typography variant="body2" sx={{ fontWeight: 700, mb: 0.75 }}>
                {entry.path}
              </Typography>
              <Stack spacing={0.75}>
                <Box>
                  <Typography variant="caption" color="text.secondary">
                    Before
                  </Typography>
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      mt: 0.5,
                      p: 1,
                      borderRadius: "4px",
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
                      fontFamily: "monospace",
                      fontSize: "0.8rem",
                      backgroundColor: alpha(theme.palette.error.main, 0.08),
                      color: theme.palette.text.primary,
                    }}
                  >
                    {formatInlineValue(entry.oldValue)}
                  </Box>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary">
                    After
                  </Typography>
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      mt: 0.5,
                      p: 1,
                      borderRadius: "4px",
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
                      fontFamily: "monospace",
                      fontSize: "0.8rem",
                      backgroundColor: alpha(theme.palette.success.main, 0.08),
                      color: theme.palette.text.primary,
                    }}
                  >
                    {formatInlineValue(entry.newValue)}
                  </Box>
                </Box>
              </Stack>
            </Box>
          ))}
        </Stack>
      </AccordionDetails>
    </Accordion>
  );
};

const DataSection = ({ label }: { label?: string }) => {
  const record = useRecordContext<AuditLogRecord>();

  void label;

  if (!record) {
    return null;
  }

  const action = String(record.action ?? "").toLowerCase();

  if (action === "create") {
    return (
      <PayloadViewer value={record.new_data} title="Created data" autoExpand />
    );
  }

  if (action === "delete") {
    return (
      <PayloadViewer value={record.old_data} title="Deleted data" autoExpand />
    );
  }

  const oldValue = normalizeForDiff(parseJsonValue(record.old_data));
  const newValue = normalizeForDiff(parseJsonValue(record.new_data));
  const diffEntries = collectDiffEntries(oldValue, newValue);

  return (
    <Stack spacing={1.5}>
      <ChangeSummary diffEntries={diffEntries} />
      <PayloadViewer value={record.old_data} title="Old data (raw)" />
      <PayloadViewer value={record.new_data} title="New data (raw)" />
    </Stack>
  );
};

export const AuditLogShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="id" />
      <DateField source="created_at" label="Time" showTime />
      <TextField source="action" />
      <TextField source="resource_type" label="Resource type" />
      <TextField source="resource_id" label="Resource ID" />
      <TextField source="actor_username" label="Username" />
      <TextField source="actor_name" label="Name" />
      <TextField source="http_method" label="HTTP method" />
      <TextField source="request_path" label="Request path" />
      <TextField source="group_status" label="Group status" />
      <TextField source="child_count" label="Related entries" />
      <DataSection label="Data" />
      <AuditLogChildren />
    </SimpleShowLayout>
  </Show>
);
