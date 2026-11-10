import { useState, type ReactNode } from "react";
import {
  CreateButton,
  EditButton,
  DeleteButton,
  RecordContextProvider,
  useCanAccess,
  useNotify,
  Title,
} from "react-admin";
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Paper,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import DragIndicator from "@mui/icons-material/DragIndicator";
import {
  DndContext,
  PointerSensor,
  KeyboardSensor,
  TouchSensor,
  useSensor,
  useSensors,
  closestCenter,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
  sortableKeyboardCoordinates,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Catalogue,
  LoungeCategory,
  loungeRequest,
  moveCatalogue,
  reorderPayload,
} from "./catalogue";
import { SoldOutButton } from "./SoldOutButton";

const catalogueKey = ["lounge", "catalogue"];

function SortableRow({
  id,
  disabled,
  label,
  children,
}: {
  id: string;
  disabled: boolean;
  label: string;
  children: ReactNode;
}) {
  const {
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
    attributes,
    listeners,
  } = useSortable({ id, disabled });
  return (
    <Box
      ref={setNodeRef}
      sx={{
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.5 : 1,
      }}
    >
      <Stack direction="row" sx={{ alignItems: "center", gap: 1 }}>
        {!disabled && (
          <IconButton
            ref={setActivatorNodeRef}
            {...attributes}
            {...listeners}
            aria-label={`Move ${label}`}
            sx={{ touchAction: "none" }}
          >
            <DragIndicator />
          </IconButton>
        )}
        <Box sx={{ flex: 1, minWidth: 0 }}>{children}</Box>
      </Stack>
    </Box>
  );
}

export function LoungeItemList() {
  const client = useQueryClient();
  const notify = useNotify();
  const { canAccess: view } = useCanAccess({
    resource: "lounge-items",
    action: "list",
  });
  const { canAccess: edit } = useCanAccess({
    resource: "lounge-items",
    action: "edit",
  });
  const { canAccess: create } = useCanAccess({
    resource: "lounge-items",
    action: "create",
  });
  const { canAccess: remove } = useCanAccess({
    resource: "lounge-items",
    action: "delete",
  });
  const query = useQuery({
    queryKey: catalogueKey,
    queryFn: () => loungeRequest<Catalogue>("lounge-catalogue"),
    enabled: Boolean(view),
  });
  const [category, setCategory] = useState<LoungeCategory | null | undefined>();
  const [name, setName] = useState("");
  const [nameSv, setNameSv] = useState("");
  const [deleting, setDeleting] = useState<LoungeCategory>();
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 200, tolerance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );
  const mutation = useMutation({
    mutationFn: ({
      path,
      method,
      body,
    }: {
      path: string;
      method: string;
      body?: unknown;
    }) => loungeRequest<unknown>(path, method, body),
    onSuccess: async () => {
      setCategory(undefined);
      setDeleting(undefined);
      await client.invalidateQueries({ queryKey: catalogueKey });
    },
    onError: async () => {
      notify(
        "Could not save catalogue changes. Reloaded the current catalogue; please try again.",
        { type: "error" },
      );
      await client.invalidateQueries({ queryKey: catalogueKey });
    },
  });
  const openCategory = (entry: LoungeCategory | null) => {
    setCategory(entry);
    setName(entry?.name ?? "");
    setNameSv(entry?.nameSv ?? "");
  };
  const saveCategory = () =>
    mutation.mutate({
      path: category ? `lounge-categories/${category.id}` : "lounge-categories",
      method: category ? "PUT" : "POST",
      body: { name, nameSv: nameSv || null },
    });
  const drop = ({ active, over }: DragEndEvent) => {
    if (
      !query.data ||
      !over ||
      mutation.isPending ||
      !edit ||
      active.id === over.id
    )
      return;
    const next = moveCatalogue(query.data, String(active.id), String(over.id));
    mutation.mutate({
      path: "lounge-catalogue",
      method: "PUT",
      body: reorderPayload(next),
    });
  };
  if (!view)
    return (
      <Alert severity="info">
        You do not have permission to view the lounge catalogue.
      </Alert>
    );
  return (
    <Box sx={{ p: 2 }}>
      <Title title="Lounge catalogue" />
      <Typography
        variant="overline"
        component="p"
        color="text.secondary"
        sx={{ mb: 1 }}
      >
        Arrange categories and items to set their order on the order page.
      </Typography>
      <Stack direction="row" sx={{ gap: 2, mb: 2 }}>
        <Typography variant="h5" sx={{ flex: 1 }}>
          Lounge catalogue
        </Typography>
        {create && (
          <>
            <Button
              onClick={() => openCategory(null)}
              disabled={mutation.isPending}
            >
              Add category
            </Button>
            <CreateButton resource="lounge-items" label="Add item" />
          </>
        )}
        <Button
          onClick={() => void query.refetch()}
          disabled={mutation.isPending}
        >
          Reload
        </Button>
      </Stack>
      {query.isPending && <Typography>Loading catalogue...</Typography>}
      {query.error && (
        <Alert severity="error">
          Could not load the catalogue. Use Reload to try again.
        </Alert>
      )}
      {query.data?.categories.length === 0 && (
        <Alert severity="info">Create a category before adding items.</Alert>
      )}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={drop}
      >
        <SortableContext
          items={
            query.data?.categories.map((entry) => `category-${entry.id}`) ?? []
          }
          strategy={verticalListSortingStrategy}
        >
          {query.data?.categories.map((entry) => {
            const items = query.data.items
              .filter((item) => item.categoryId === entry.id)
              .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);
            return (
              <Paper key={entry.id} sx={{ p: 2, mb: 2 }}>
                <SortableRow
                  id={`category-${entry.id}`}
                  disabled={!edit || mutation.isPending}
                  label={entry.name}
                >
                  <Stack direction="row" sx={{ alignItems: "center" }}>
                    <Typography variant="h6" sx={{ flex: 1 }}>
                      {entry.name}
                    </Typography>
                    {edit && (
                      <Button
                        disabled={mutation.isPending}
                        onClick={() => openCategory(entry)}
                      >
                        Edit category
                      </Button>
                    )}
                    {remove && (
                      <Button
                        color="error"
                        disabled={mutation.isPending}
                        onClick={() => setDeleting(entry)}
                      >
                        Delete category
                      </Button>
                    )}
                  </Stack>
                  <SortableContext
                    items={items.map((item) => `item-${item.id}`)}
                    strategy={verticalListSortingStrategy}
                  >
                    {items.map((item) => (
                      <SortableRow
                        key={item.id}
                        id={`item-${item.id}`}
                        disabled={!edit || mutation.isPending}
                        label={item.name}
                      >
                        <RecordContextProvider value={item}>
                          <Stack
                            direction="row"
                            sx={{ alignItems: "center", flexWrap: "wrap" }}
                          >
                            <Typography sx={{ flex: 1 }}>
                              {item.name}
                            </Typography>
                            <SoldOutButton />
                            {edit && <EditButton resource="lounge-items" />}
                            {remove && (
                              <DeleteButton
                                resource="lounge-items"
                                mutationMode="pessimistic"
                                mutationOptions={{
                                  onSuccess: () =>
                                    client.invalidateQueries({
                                      queryKey: catalogueKey,
                                    }),
                                }}
                              />
                            )}
                          </Stack>
                        </RecordContextProvider>
                      </SortableRow>
                    ))}
                    {!items.length && (
                      <Typography sx={{ p: 2 }}>Drop items here</Typography>
                    )}
                  </SortableContext>
                </SortableRow>
              </Paper>
            );
          })}
        </SortableContext>
      </DndContext>
      <Dialog
        open={category !== undefined}
        onClose={() => !mutation.isPending && setCategory(undefined)}
      >
        <DialogTitle>{category ? "Edit category" : "Add category"}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            label="Name (English)"
            value={name}
            onChange={(event) => setName(event.target.value)}
            fullWidth
            margin="dense"
            required
          />
          <TextField
            label="Name (Swedish)"
            value={nameSv}
            onChange={(event) => setNameSv(event.target.value)}
            fullWidth
            margin="dense"
          />
        </DialogContent>
        <DialogActions>
          <Button
            disabled={mutation.isPending}
            onClick={() => setCategory(undefined)}
          >
            Cancel
          </Button>
          <Button
            disabled={mutation.isPending || !name.trim()}
            onClick={saveCategory}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={Boolean(deleting)}
        onClose={() => !mutation.isPending && setDeleting(undefined)}
      >
        <DialogTitle>Delete category and its items?</DialogTitle>
        <DialogContent>
          {deleting?.name}: this will delete{" "}
          {query.data?.items.filter((item) => item.categoryId === deleting?.id)
            .length ?? 0}{" "}
          items. Saved order history will remain unchanged.
        </DialogContent>
        <DialogActions>
          <Button
            disabled={mutation.isPending}
            onClick={() => setDeleting(undefined)}
          >
            Cancel
          </Button>
          <Button
            color="error"
            disabled={mutation.isPending}
            onClick={() =>
              deleting &&
              mutation.mutate({
                path: `lounge-categories/${deleting.id}`,
                method: "DELETE",
              })
            }
          >
            Delete category and items
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
