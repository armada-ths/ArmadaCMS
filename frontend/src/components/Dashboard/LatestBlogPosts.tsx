import { useGetList } from "react-admin";
import { useNavigate } from "react-router";
import {
  Box,
  Card,
  CardContent,
  Divider,
  List,
  ListItem,
  ListItemText,
  Skeleton,
  Stack,
  Typography,
} from "@mui/material";
import { Newspaper } from "@mui/icons-material";
import { hasPerm } from "../../utils/permissions";

interface BlogPost {
  id: number;
  title: string;
  author: string;
  createdAt: string;
}

export const LatestBlogPosts = ({ perms }: { perms: string[] }) => {
  const navigate = useNavigate();
  const { data, isPending } = useGetList<BlogPost>(
    "blogposts",
    {
      sort: { field: "created_at", order: "DESC" },
      pagination: { page: 1, perPage: 3 },
    },
    { enabled: hasPerm(perms, "blogposts.list") },
  );

  if (!hasPerm(perms, "blogposts.list")) return null;

  return (
    <Card variant="outlined" sx={{ height: "100%" }}>
      <CardContent>
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          mb={1}
        >
          <Stack direction="row" alignItems="center" spacing={1}>
            <Newspaper color="action" />
            <Typography variant="h6">Latest blog posts</Typography>
          </Stack>
          <Typography
            variant="body2"
            color="primary"
            sx={{
              cursor: "pointer",
              "&:hover": { textDecoration: "underline" },
            }}
            onClick={() => navigate("/admin/blogposts")}
          >
            View all →
          </Typography>
        </Stack>
        <Divider />
        {isPending ? (
          <Stack spacing={1} mt={1}>
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} height={56} />
            ))}
          </Stack>
        ) : (data ?? []).length === 0 ? (
          <Typography variant="body2" color="text.secondary" mt={1}>
            No blog posts yet.
          </Typography>
        ) : (
          <List dense disablePadding>
            {(data ?? []).map((post, idx) => (
              <Box key={post.id}>
                <ListItem
                  disableGutters
                  sx={{ cursor: "pointer" }}
                  onClick={() => navigate(`/admin/blogposts/${post.id}`)}
                >
                  <ListItemText
                    primary={post.title || "(Untitled)"}
                    secondary={`${post.author} · ${new Date(post.createdAt).toLocaleDateString()}`}
                    slotProps={{
                      primary: { variant: "body2", fontWeight: 600 },
                      secondary: { variant: "caption" },
                    }}
                  />
                </ListItem>
                {idx < (data?.length ?? 0) - 1 && <Divider component="li" />}
              </Box>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  );
};
