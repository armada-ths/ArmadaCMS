import { useState, useCallback, useEffect } from "react";
import { useGetList } from "react-admin";
import { useNavigate } from "react-router";
import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  Grid,
  Skeleton,
  Stack,
  Typography,
} from "@mui/material";
import {
  AccountCircle,
  AdminPanelSettings,
  Badge,
  Business,
  CalendarMonth,
  Category,
  DateRange,
  Event,
  Group,
  Person,
  School,
  Stars,
  WorkHistory,
} from "@mui/icons-material";
import { type SvgIconComponent } from "@mui/icons-material";
import { hasPerm } from "../../utils/permissions";

// ─── Resource catalog (priority-ordered) ─────────────────────────────────────

interface ResourceConfig {
  resource: string;
  label: string;
  Icon: SvgIconComponent;
  permission: string;
  to: string;
}

const ALL_RESOURCES: ResourceConfig[] = [
  {
    resource: "exhibitors",
    label: "Exhibitors",
    Icon: Business,
    permission: "exhibitors.list",
    to: "/admin/exhibitors",
  },
  {
    resource: "events",
    label: "Events",
    Icon: Event,
    permission: "events.list",
    to: "/admin/events",
  },
  {
    resource: "customusers",
    label: "Users",
    Icon: Person,
    permission: "customusers.list",
    to: "/admin/customusers",
  },
  {
    resource: "highlightcards",
    label: "Highlight cards",
    Icon: Stars,
    permission: "highlightcards.list",
    to: "/admin/highlightcards",
  },
  {
    resource: "fairdates",
    label: "Fair dates",
    Icon: CalendarMonth,
    permission: "fairdates.list",
    to: "/admin/fairdates",
  },
  {
    resource: "recruitmentperiods",
    label: "Recruitment periods",
    Icon: DateRange,
    permission: "recruitmentperiods.list",
    to: "/admin/recruitmentperiods",
  },
  {
    resource: "recruitmentroles",
    label: "Recruitment roles",
    Icon: Badge,
    permission: "recruitmentroles.list",
    to: "/admin/recruitmentroles",
  },
  {
    resource: "profiles",
    label: "Profiles",
    Icon: AccountCircle,
    permission: "profiles.list",
    to: "/admin/profiles",
  },
  {
    resource: "teams",
    label: "Teams",
    Icon: Group,
    permission: "teams.list",
    to: "/admin/teams",
  },
  {
    resource: "programs",
    label: "Programs",
    Icon: School,
    permission: "programs.list",
    to: "/admin/programs",
  },
  {
    resource: "industries",
    label: "Industries",
    Icon: Category,
    permission: "industries.list",
    to: "/admin/industries",
  },
  {
    resource: "employments",
    label: "Employments",
    Icon: WorkHistory,
    permission: "employments.list",
    to: "/admin/employments",
  },
  {
    resource: "roles",
    label: "Roles",
    Icon: AdminPanelSettings,
    permission: "roles.list",
    to: "/admin/roles",
  },
];

// ─── Count fetcher (headless) ─────────────────────────────────────────────────

const CountFetcher = ({
  resource,
  onCount,
}: {
  resource: string;
  onCount: (resource: string, total: number) => void;
}) => {
  const { total, isPending } = useGetList(resource, {
    pagination: { page: 1, perPage: 1 },
    sort: { field: "id", order: "ASC" },
  });

  useEffect(() => {
    if (!isPending && total !== undefined) {
      onCount(resource, total);
    }
  }, [total, isPending, resource, onCount]);

  return null;
};

// ─── Resource card (presentational) ──────────────────────────────────────────

const ResourceCard = ({
  label,
  Icon,
  to,
  total,
}: {
  label: string;
  Icon: SvgIconComponent;
  to: string;
  total: number;
}) => {
  const navigate = useNavigate();
  return (
    <Grid size={{ xs: 12, sm: 6, md: 4 }}>
      <Card variant="outlined" sx={{ height: "100%" }}>
        <CardActionArea
          onClick={() => navigate(to)}
          sx={{ height: "100%", p: 0.5 }}
        >
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1.5}>
              <Icon color="primary" fontSize="large" />
              <Box flexGrow={1}>
                <Typography variant="h6" component="div" lineHeight={1.2}>
                  {label}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {total} {total === 1 ? "entry" : "entries"}
                </Typography>
              </Box>
            </Stack>
          </CardContent>
        </CardActionArea>
      </Card>
    </Grid>
  );
};

// ─── Resource grid ────────────────────────────────────────────────────────────

export const ResourceGrid = ({
  perms,
  permsPending,
}: {
  perms: string[];
  permsPending: boolean;
}) => {
  const [resourceCounts, setResourceCounts] = useState<Record<string, number>>(
    {},
  );

  const handleResourceCount = useCallback((resource: string, total: number) => {
    setResourceCounts((prev) => ({ ...prev, [resource]: total }));
  }, []);

  const permittedResources = permsPending
    ? []
    : ALL_RESOURCES.filter((r) => hasPerm(perms, r.permission));

  const countsLoaded = permittedResources.every(
    (r) => resourceCounts[r.resource] !== undefined,
  );

  const visibleResources = countsLoaded
    ? permittedResources
        .filter((r) => (resourceCounts[r.resource] ?? 0) > 0)
        .slice(0, 6)
    : [];

  return (
    <>
      {permittedResources.map((r) => (
        <CountFetcher
          key={r.resource}
          resource={r.resource}
          onCount={handleResourceCount}
        />
      ))}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        {permsPending || !countsLoaded
          ? Array.from({ length: 6 }).map((_, i) => (
              <Grid key={i} size={{ xs: 12, sm: 6, md: 4 }}>
                <Skeleton
                  variant="rectangular"
                  height={80}
                  sx={{ borderRadius: 1 }}
                />
              </Grid>
            ))
          : visibleResources.map((r) => (
              <ResourceCard
                key={r.resource}
                label={r.label}
                Icon={r.Icon}
                to={r.to}
                total={resourceCounts[r.resource] ?? 0}
              />
            ))}
      </Grid>
    </>
  );
};
