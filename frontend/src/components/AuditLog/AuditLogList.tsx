import { ResourceList } from "../ResourceList";
import { useId, useState } from "react";
import {
  Button,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
} from "@mui/material";
import ArrowDropDownIcon from "@mui/icons-material/ArrowDropDown";
import CheckBoxIcon from "@mui/icons-material/CheckBox";
import CheckBoxOutlineBlankIcon from "@mui/icons-material/CheckBoxOutlineBlank";
import {
  Datagrid,
  TextField,
  DateField,
  ShowButton,
  TextInput,
  SelectInput,
  FilterButton,
  TopToolbar,
  ListProps,
  useListContext,
} from "react-admin";
import { AuditLogChildren } from "./AuditLogChildren";

const auditLogFilters = [
  <SelectInput
    key="action"
    source="action"
    choices={[
      { id: "create", name: "Create" },
      { id: "update", name: "Update" },
      { id: "delete", name: "Delete" },
      { id: "sync", name: "Sync" },
    ]}
  />,
  <TextInput
    key="resource_type"
    source="resource_type"
    label="Resource type"
  />,
  <TextInput key="actor_username" source="actor_username" label="Username" />,
];

const ListActions = () => {
  const { displayedFilters, filterValues, setFilters } = useListContext();
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const menuId = useId();
  const buttonId = useId();
  const activityOptions = [
    { key: "include_auth", label: "Show login activity" },
    { key: "include_guest_photos", label: "Show guest photo activity" },
  ] as const;
  const activeCount = activityOptions.filter(
    ({ key }) => filterValues[key] === "true",
  ).length;

  const handleToggle = (key: (typeof activityOptions)[number]["key"]) => {
    const nextFilters = { ...filterValues };
    if (nextFilters[key] === "true") {
      delete nextFilters[key];
    } else {
      nextFilters[key] = "true";
    }
    setFilters(nextFilters, displayedFilters);
  };

  return (
    <TopToolbar>
      <Button
        id={buttonId}
        size="small"
        aria-controls={anchorEl ? menuId : undefined}
        aria-haspopup="menu"
        aria-expanded={anchorEl ? true : undefined}
        onClick={(event) => setAnchorEl(event.currentTarget)}
        endIcon={<ArrowDropDownIcon />}
      >
        Activity{activeCount > 0 ? ` (${activeCount})` : ""}
      </Button>
      <Menu
        id={menuId}
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        slotProps={{ list: { "aria-labelledby": buttonId } }}
      >
        {activityOptions.map(({ key, label }) => {
          const checked = filterValues[key] === "true";
          return (
            <MenuItem
              key={key}
              role="menuitemcheckbox"
              aria-checked={checked}
              onClick={() => handleToggle(key)}
            >
              <ListItemIcon>
                {checked ? (
                  <CheckBoxIcon fontSize="small" />
                ) : (
                  <CheckBoxOutlineBlankIcon fontSize="small" />
                )}
              </ListItemIcon>
              <ListItemText>{label}</ListItemText>
            </MenuItem>
          );
        })}
      </Menu>
      <FilterButton />
    </TopToolbar>
  );
};

export const AuditLogList = (props: ListProps) => (
  <ResourceList
    {...props}
    sort={{ field: "created_at", order: "DESC" }}
    filters={auditLogFilters}
    actions={<ListActions />}
    sx={{
      minWidth: 0,
      maxWidth: { xs: "calc(100vw - 16px)", sm: "none" },
      "& .RaList-main": { minWidth: 0 },
      "& .RaList-content": { minWidth: 0, overflowX: "auto" },
    }}
  >
    <Datagrid
      bulkActionButtons={false}
      rowClick="show"
      expand={<AuditLogChildren />}
      isRowExpandable={(record) => Number(record.child_count ?? 0) > 0}
    >
      <DateField source="created_at" label="Time" showTime />
      <TextField source="action" />
      <TextField source="resource_type" label="Resource" />
      <TextField source="resource_id" label="Resource ID" />
      <TextField source="actor_username" label="User" />
      <TextField source="http_method" label="Method" />
      <TextField source="request_path" label="Path" />
      <TextField source="child_count" label="Related" />
      <ShowButton />
    </Datagrid>
  </ResourceList>
);
