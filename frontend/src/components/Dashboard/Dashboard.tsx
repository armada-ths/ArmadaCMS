import { useGetIdentity, usePermissions, Title } from "react-admin";
import {
  Box,
  Card,
  CardContent,
  Grid,
  Skeleton,
  Typography,
} from "@mui/material";
import { hasPerm } from "../../utils/permissions";
import { ResourceGrid } from "./ResourceGrid";
import { RecentActivity } from "./RecentActivity";
import { LatestBlogPosts } from "./LatestBlogPosts";
import { FeatureFlagsSummary } from "./FeatureFlagsSummary";
import { EventroSyncCard } from "./EventroSyncCard";

export const Dashboard = () => {
  const { data: identity, isPending: identityPending } = useGetIdentity();
  const { permissions, isPending: permsPending } = usePermissions();

  const perms: string[] = Array.isArray(permissions) ? permissions : [];

  const today = new Date().toLocaleDateString(undefined, {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  return (
    <Box p={3}>
      <Title title="Dashboard" />

      {/* Welcome */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h5" gutterBottom>
            {identityPending ? (
              <Skeleton width={280} />
            ) : (
              <>Welcome back, {identity?.fullName}!</>
            )}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {today}
          </Typography>
        </CardContent>
      </Card>

      {/* Resource cards */}
      <Typography variant="h6" gutterBottom>
        Resources
      </Typography>
      <ResourceGrid perms={perms} permsPending={permsPending} />

      {/* Activity + Blog posts */}
      {!permsPending && (
        <Grid container spacing={2} alignItems="flex-start">
          {hasPerm(perms, "auditlogs.list") && (
            <Grid
              size={{
                xs: 12,
                md: hasPerm(perms, "blogposts.list") ? 7 : 12,
              }}
            >
              <RecentActivity perms={perms} />
            </Grid>
          )}
          {hasPerm(perms, "blogposts.list") && (
            <Grid
              size={{
                xs: 12,
                md: hasPerm(perms, "auditlogs.list") ? 5 : 12,
              }}
            >
              <LatestBlogPosts perms={perms} />
            </Grid>
          )}
        </Grid>
      )}

      {/* Feature flags */}
      {!permsPending && <FeatureFlagsSummary perms={perms} />}

      {/* Eventro sync */}
      {!permsPending && <EventroSyncCard perms={perms} />}
    </Box>
  );
};
