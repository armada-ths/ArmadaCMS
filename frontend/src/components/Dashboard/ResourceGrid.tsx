import { useBasename, useGetList } from "react-admin";
import { Link } from "react-router-dom";
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
import type { NavigationNode, ResourceLink } from "../../navigation/catalog";
import { useNavigation } from "../../navigation/useNavigation";

const ResourceCard = ({ node }: { node: ResourceLink }) => {
  const basename = useBasename();
  const { total, isPending, error } = useGetList(node.resource, {
    pagination: { page: 1, perPage: 1 },
    sort: { field: "id", order: "ASC" },
  });
  const { Icon } = node;
  return (
    <Grid size={{ xs: 12, sm: 6, md: 4 }}>
      <Card variant="outlined" sx={{ height: "100%" }}>
        <CardActionArea
          component={Link}
          to={`${basename}${node.to}`}
          sx={{ height: "100%", p: 0.5 }}
        >
          <CardContent>
            <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
              <Icon color="primary" fontSize="large" />
              <Box sx={{ flexGrow: 1 }}>
                <Typography
                  variant="h6"
                  component="h4"
                  sx={{ lineHeight: 1.2 }}
                >
                  {node.label}
                </Typography>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  component="div"
                >
                  {error ? (
                    "Count unavailable"
                  ) : isPending ? (
                    <Skeleton width={80} />
                  ) : total === undefined ? (
                    "Count unavailable"
                  ) : (
                    `${total} ${total === 1 ? "entry" : "entries"}`
                  )}
                </Typography>
              </Box>
            </Stack>
          </CardContent>
        </CardActionArea>
      </Card>
    </Grid>
  );
};

const ResourceSection = ({
  nodes,
  depth = 0,
}: {
  nodes: readonly NavigationNode[];
  depth?: number;
}) => (
  <Grid container spacing={2} sx={{ mb: 3 }}>
    {nodes.map((node) => {
      if (node.kind === "dashboard") return null;
      if (node.kind === "resource")
        return <ResourceCard key={node.id} node={node} />;
      return (
        <Grid key={node.id} size={12}>
          <Box component="section" aria-labelledby={`resources-${node.id}`}>
            <Typography
              id={`resources-${node.id}`}
              variant={depth === 0 ? "h6" : "subtitle1"}
              component={depth === 0 ? "h3" : "h4"}
              sx={{ mb: 1, fontWeight: 600 }}
            >
              {node.label}
            </Typography>
            <ResourceSection nodes={node.children} depth={depth + 1} />
          </Box>
        </Grid>
      );
    })}
  </Grid>
);

export const ResourceGrid = () => {
  const { nodes, isPending, error } = useNavigation();
  if (isPending)
    return (
      <Skeleton
        variant="rectangular"
        height={80}
        sx={{ mb: 3, borderRadius: 1 }}
      />
    );
  if (error)
    return (
      <Typography color="error" sx={{ mb: 3 }}>
        Resources unavailable
      </Typography>
    );
  return <ResourceSection nodes={nodes} />;
};
