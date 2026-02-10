import { Chip, Tooltip } from "@mui/material";
import { useRecordContext } from "react-admin";

export const AutoOverrideBadge = () => {
  const record = useRecordContext();

  const autoValue = record?.autoValue as boolean | null | undefined;
  const autoUpdatedAt = record?.autoUpdatedAt as string | null | undefined;
  const enabled = record?.enabled as boolean | undefined;

  const isAutoCapable =
    Boolean(autoUpdatedAt) && autoValue !== null && autoValue !== undefined;
  if (!isAutoCapable) {
    return null;
  }

  const isAutoAssigned = enabled === autoValue;
  const label = isAutoAssigned ? "Auto override" : "Manual override";
  const tooltipTitle = autoUpdatedAt
    ? `Last auto update: ${new Date(autoUpdatedAt).toLocaleString()}`
    : "No auto updates yet";

  return (
    <Tooltip title={tooltipTitle} arrow>
      <Chip
        label={label}
        color={isAutoAssigned ? "info" : "warning"}
        size="small"
      />
    </Tooltip>
  );
};
