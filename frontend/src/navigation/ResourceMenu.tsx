import { useState } from "react";
import { Menu, MenuItemLink, useBasename, useSidebarState } from "react-admin";
import { useLocation } from "react-router-dom";
import {
  Box,
  Collapse,
  MenuItem,
  ListItemIcon,
  Typography,
  Tooltip,
} from "@mui/material";
import ExpandLess from "@mui/icons-material/ExpandLess";
import ExpandMore from "@mui/icons-material/ExpandMore";
import {
  getActiveGroupIds,
  isNavigationActive,
  type NavigationNode,
} from "./catalog";
import { useNavigation } from "./useNavigation";

export const ResourceMenu = () => {
  const { nodes } = useNavigation();
  const { pathname } = useLocation();
  const basename = useBasename();
  const [sidebarOpen, setSidebarOpen] = useSidebarState();
  const activeGroups = Object.fromEntries(
    getActiveGroupIds(nodes, pathname, basename).map((id) => [id, true]),
  );
  const [state, setState] = useState<{
    pathname: string;
    open: Record<string, boolean>;
  }>({ pathname, open: activeGroups });

  // Preserve manual choices, opening active ancestors when navigation changes.
  if (state.pathname !== pathname) {
    setState({ pathname, open: { ...state.open, ...activeGroups } });
  }
  const openGroups =
    state.pathname === pathname
      ? state.open
      : { ...state.open, ...activeGroups };

  const renderNode = (node: NavigationNode, depth = 0) => {
    const { Icon } = node;
    const active = isNavigationActive(node, pathname, basename);
    const itemSx = {
      textAlign: "left",
      pl: sidebarOpen ? 2 + depth * 2 : 2,
      color: active ? "text.primary" : "text.secondary",
      "& .MuiListItemIcon-root": { minWidth: 40, color: "text.secondary" },
      "&.Mui-focusVisible": {
        outline: "2px solid",
        outlineColor: "primary.main",
        outlineOffset: -2,
      },
    };
    if (node.kind !== "group") {
      return (
        <MenuItemLink
          key={node.id}
          to={`${basename}${node.to}`}
          primaryText={node.label}
          leftIcon={<Icon />}
          selected={active}
          aria-current={active ? "page" : undefined}
          sx={itemSx}
        />
      );
    }
    // Active groups also open when permissions finish loading on a direct link.
    const expanded = sidebarOpen && (openGroups[node.id] ?? active);
    const submenuId = `navigation-${node.id}`;
    return (
      <Box component="li" role="none" key={node.id} sx={{ listStyle: "none" }}>
        <Tooltip title={sidebarOpen ? "" : node.label} placement="right">
          <MenuItem
            component="button"
            type="button"
            tabIndex={0}
            role="menuitem"
            aria-label={node.label}
            aria-expanded={expanded}
            aria-controls={submenuId}
            selected={active}
            onClick={() => {
              setSidebarOpen(true);
              setState((previous) => ({
                pathname,
                open: {
                  ...previous.open,
                  [node.id]: !sidebarOpen || !expanded,
                },
              }));
            }}
            sx={{ ...itemSx, width: "100%" }}
          >
            <ListItemIcon>
              <Icon />
            </ListItemIcon>
            {sidebarOpen && (
              <>
                <Typography variant="inherit" noWrap sx={{ flexGrow: 1 }}>
                  {node.label}
                </Typography>
                {expanded ? <ExpandLess /> : <ExpandMore />}
              </>
            )}
          </MenuItem>
        </Tooltip>
        <Collapse in={expanded}>
          <Box
            component="ul"
            role="menu"
            aria-label={node.label}
            id={submenuId}
            sx={{ p: 0, m: 0, listStyle: "none" }}
          >
            {node.children.map((child) => renderNode(child, depth + 1))}
          </Box>
        </Collapse>
      </Box>
    );
  };

  return <Menu>{nodes.map((node) => renderNode(node))}</Menu>;
};
