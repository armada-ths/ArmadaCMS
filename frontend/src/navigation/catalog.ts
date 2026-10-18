import AccountCircle from "@mui/icons-material/AccountCircle";
import AdminPanelSettings from "@mui/icons-material/AdminPanelSettings";
import Badge from "@mui/icons-material/Badge";
import Business from "@mui/icons-material/Business";
import CalendarMonth from "@mui/icons-material/CalendarMonth";
import Category from "@mui/icons-material/Category";
import Dashboard from "@mui/icons-material/Dashboard";
import DateRange from "@mui/icons-material/DateRange";
import Event from "@mui/icons-material/Event";
import Flag from "@mui/icons-material/Flag";
import Group from "@mui/icons-material/Group";
import History from "@mui/icons-material/History";
import Article from "@mui/icons-material/Article";
import Person from "@mui/icons-material/Person";
import PhotoLibrary from "@mui/icons-material/PhotoLibrary";
import School from "@mui/icons-material/School";
import Stars from "@mui/icons-material/Stars";
import Timeline from "@mui/icons-material/Timeline";
import Web from "@mui/icons-material/Web";
import WorkHistory from "@mui/icons-material/WorkHistory";
import type { SvgIconComponent } from "@mui/icons-material";

export const RESOURCE_LABELS = {
  photoevents: "Guest photos",
  exhibitors: "Exhibitors",
  industries: "Industries",
  programs: "Programs",
  employments: "Employments",
  events: "Events",
  fairdates: "Fair dates",
  blogposts: "Blog posts",
  highlightcards: "Highlight cards",
  "timeline-entries": "History timeline entries",
  "timeline-eras": "History timeline eras",
  profiles: "Profiles",
  teams: "Teams",
  recruitmentperiods: "Recruitment periods",
  recruitmentroles: "Recruitment roles",
  customusers: "Users",
  roles: "Roles",
  featureflags: "Feature flags",
  auditlogs: "Audit logs",
} as const;

export type ResourceName = keyof typeof RESOURCE_LABELS;
interface NavigationBase {
  id: string;
  label: string;
  Icon: SvgIconComponent;
}
export interface ResourceLink extends NavigationBase {
  kind: "resource";
  resource: ResourceName;
  to: string;
}
export interface NavigationGroup extends NavigationBase {
  kind: "group";
  children: readonly NavigationNode[];
}
export interface DashboardLink extends NavigationBase {
  kind: "dashboard";
  to: string;
}
export type NavigationNode = ResourceLink | NavigationGroup | DashboardLink;

const resource = (
  name: ResourceName,
  Icon: SvgIconComponent,
  label: string = RESOURCE_LABELS[name],
): ResourceLink => ({
  kind: "resource",
  id: name,
  resource: name,
  label,
  Icon,
  to: `/${name}`,
});

export const NAVIGATION: readonly NavigationNode[] = [
  {
    kind: "dashboard",
    id: "dashboard",
    label: "Dashboard",
    Icon: Dashboard,
    to: "/",
  },
  resource("photoevents", PhotoLibrary),
  {
    kind: "group",
    id: "fair",
    label: "Fair & events",
    Icon: Event,
    children: [
      {
        kind: "group",
        id: "exhibitor-settings",
        label: "Exhibitors",
        Icon: Business,
        children: [
          resource("exhibitors", Business, "All exhibitors"),
          resource("industries", Category),
          resource("programs", School),
          resource("employments", WorkHistory),
        ],
      },
      resource("events", Event),
      resource("fairdates", CalendarMonth),
    ],
  },
  {
    kind: "group",
    id: "website",
    label: "Website content",
    Icon: Web,
    children: [
      resource("blogposts", Article),
      resource("highlightcards", Stars),
      {
        kind: "group",
        id: "history",
        label: "History timeline",
        Icon: History,
        children: [
          resource("timeline-entries", Timeline, "Entries"),
          resource("timeline-eras", DateRange, "Eras"),
        ],
      },
    ],
  },
  {
    kind: "group",
    id: "organisation",
    label: "Organisation",
    Icon: Group,
    children: [resource("profiles", AccountCircle), resource("teams", Group)],
  },
  {
    kind: "group",
    id: "recruitment",
    label: "Recruitment",
    Icon: Badge,
    children: [
      resource("recruitmentperiods", DateRange),
      resource("recruitmentroles", Badge),
    ],
  },
  {
    kind: "group",
    id: "administration",
    label: "Administration",
    Icon: AdminPanelSettings,
    children: [
      resource("customusers", Person),
      resource("roles", AdminPanelSettings),
      resource("featureflags", Flag),
      resource("auditlogs", History),
    ],
  },
];

export const getResourceLinks = (
  nodes: readonly NavigationNode[],
): ResourceLink[] =>
  nodes.flatMap((node) =>
    node.kind === "group"
      ? getResourceLinks(node.children)
      : node.kind === "resource"
        ? [node]
        : [],
  );

export const RESOURCE_NAMES = getResourceLinks(NAVIGATION).map(
  (node) => node.resource,
);

export const filterNavigation = (
  nodes: readonly NavigationNode[],
  canAccess: Record<string, boolean>,
): NavigationNode[] =>
  nodes.flatMap((node): NavigationNode[] => {
    if (node.kind === "group") {
      const children = filterNavigation(node.children, canAccess);
      return children.length ? [{ ...node, children }] : [];
    }
    return node.kind === "dashboard" || canAccess[node.resource] === true
      ? [node]
      : [];
  });

export const isNavigationActive = (
  node: NavigationNode,
  pathname: string,
  basename: string,
): boolean => {
  if (node.kind === "group")
    return node.children.some((child) =>
      isNavigationActive(child, pathname, basename),
    );
  const path = `${basename}${node.to}`;
  return node.kind === "dashboard"
    ? pathname === path || pathname === basename
    : pathname === path || pathname.startsWith(`${path}/`);
};

export const getActiveGroupIds = (
  nodes: readonly NavigationNode[],
  pathname: string,
  basename: string,
): string[] =>
  nodes.flatMap((node) =>
    node.kind === "group"
      ? [
          ...(isNavigationActive(node, pathname, basename) ? [node.id] : []),
          ...getActiveGroupIds(node.children, pathname, basename),
        ]
      : [],
  );
