import { useState } from "react";
import {
  List,
  Datagrid,
  TextField,
  DateField,
  BooleanField,
  ShowButton,
  Show,
  TopToolbar,
  ListButton,
  SimpleShowLayout,
  ArrayField,
  NumberField,
  SelectInput,
  useListContext,
  useRecordContext,
  useCanAccess,
  useNotify,
  useRefresh,
  useUpdate,
} from "react-admin";
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Select,
  Typography,
} from "@mui/material";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { yellow } from "@mui/material/colors";
import { loungeRequest } from "../LoungeItem/catalogue";

interface OrderRecord {
  id: number;
  status: string;
  deliveryStatus: string;
  isTest: boolean;
}
interface FilterOptions {
  companies: { id: number; name: string }[];
  categories: { id: number; name: string }[];
  items: { id: number; name: string }[];
}
interface Total {
  itemId: number;
  categoryId: number;
  itemName: string;
  categoryName: string;
  quantity: string;
}
const statuses = ["pending", "completed"].map((id) => ({
  id,
  name: id,
}));
const emptyFilterOption = {
  emptyText: "(none)",
  slotProps: {
    select: { displayEmpty: true },
    inputLabel: { shrink: true },
  },
};

function OrderStatus({ value }: { value: string }) {
  const color = value === "completed" ? "success" : "warning";
  return (
    <Chip
      label={value}
      color={color}
      size="small"
      sx={{
        fontWeight: 700,
        ...(value === "pending" && {
          bgcolor: yellow[700],
          color: "common.black",
        }),
      }}
    />
  );
}

function OrderStatusField({
  source = "status",
}: {
  source?: string;
  label?: string;
}) {
  const record = useRecordContext<Record<string, string>>();
  if (!record) return null;
  return statuses.some((status) => status.id === record[source]) ? (
    <OrderStatus value={record[source]} />
  ) : (
    <TextField source={source} />
  );
}

function OrderTotals() {
  const { filterValues } = useListContext();
  const query = useQuery({
    queryKey: ["lounge", "totals", filterValues],
    queryFn: () =>
      loungeRequest<Total[]>(
        `lounge-orders/totals?filter=${encodeURIComponent(JSON.stringify(filterValues))}`,
      ),
  });
  const categoryTotals = new Map<
    number,
    { name: string; quantity: bigint; items: Total[] }
  >();
  for (const total of query.data ?? []) {
    const previous = categoryTotals.get(total.categoryId);
    categoryTotals.set(total.categoryId, {
      name: total.categoryName,
      quantity: (previous?.quantity ?? 0n) + BigInt(total.quantity),
      items: [...(previous?.items ?? []), total],
    });
  }
  return (
    <Box sx={{ my: 2 }}>
      <Typography variant="h6">
        Quantity totals (all matching orders)
      </Typography>
      {query.isPending && <Typography>Loading totals...</Typography>}
      {query.error && (
        <Alert severity="error">
          Could not load totals.{" "}
          <Button onClick={() => void query.refetch()}>Retry</Button>
        </Alert>
      )}
      {query.data?.length === 0 && <Typography>No matching items.</Typography>}
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: {
            xs: "minmax(0, 1fr)",
            sm: "repeat(auto-fit, minmax(240px, 1fr))",
          },
          gap: 2,
          mt: 1,
        }}
      >
        {[...categoryTotals].map(([id, category]) => (
          <Box
            component="section"
            aria-labelledby={`lounge-total-category-${id}`}
            key={id}
            sx={{
              border: 1,
              borderColor: "divider",
              borderRadius: 1,
              p: 2,
              minWidth: 0,
            }}
          >
            <Typography
              component="h3"
              variant="subtitle1"
              id={`lounge-total-category-${id}`}
              sx={{
                display: "flex",
                justifyContent: "space-between",
                gap: 2,
                fontWeight: 700,
                pb: 1,
                borderBottom: 1,
                borderColor: "divider",
              }}
            >
              <Box component="span" sx={{ overflowWrap: "anywhere" }}>
                {category.name}
              </Box>
              <Box component="span" sx={{ fontVariantNumeric: "tabular-nums" }}>
                {category.quantity.toString()}
              </Box>
            </Typography>
            <Box component="dl" sx={{ m: 0, pt: 1 }}>
              {category.items.map((item) => (
                <Box
                  component="div"
                  key={`${item.itemId}-${item.itemName}`}
                  sx={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "baseline",
                    gap: 2,
                    py: 0.5,
                  }}
                >
                  <Typography
                    component="dt"
                    variant="body2"
                    sx={{ overflowWrap: "anywhere" }}
                  >
                    {item.itemName}
                  </Typography>
                  <Typography
                    component="dd"
                    variant="body2"
                    sx={{ m: 0, fontVariantNumeric: "tabular-nums" }}
                  >
                    {item.quantity}
                  </Typography>
                </Box>
              ))}
            </Box>
          </Box>
        ))}
      </Box>
    </Box>
  );
}

export function LoungeOrderActions() {
  const order = useRecordContext<OrderRecord>();
  const { canAccess } = useCanAccess({
    resource: "lounge-orders",
    action: "edit",
  });
  const [update, { isPending }] = useUpdate();
  const [confirm, setConfirm] = useState(false);
  const notify = useNotify();
  const refresh = useRefresh();
  const client = useQueryClient();
  const reload = () => {
    refresh();
    void client.invalidateQueries({ queryKey: ["lounge"] });
  };
  const retry = useMutation({
    mutationFn: () => loungeRequest(`lounge-orders/${order?.id}/retry`, "POST"),
    onSuccess: () => {
      setConfirm(false);
      notify("Notification retry processed. Check its delivery status.", {
        type: "info",
      });
      reload();
    },
    onError: () => {
      setConfirm(false);
      notify(
        "Could not retry notification. Refresh order details before trying again.",
        { type: "error" },
      );
      reload();
    },
  });
  if (!order || !canAccess) return null;
  return (
    <Box sx={{ display: "flex", gap: 2, my: 2 }}>
      <Select
        inputProps={{ "aria-label": "Order status" }}
        value={order.status}
        renderValue={(value) => <OrderStatus value={value} />}
        disabled={isPending || retry.isPending}
        onChange={(event) =>
          update(
            "lounge-orders",
            {
              id: order.id,
              data: { status: event.target.value },
              previousData: order,
            },
            {
              mutationMode: "pessimistic",
              onSuccess: reload,
              onError: () =>
                notify("Could not change order status.", { type: "error" }),
            },
          )
        }
      >
        {statuses.map((status) => (
          <MenuItem key={status.id} value={status.id}>
            <OrderStatus value={status.id} />
          </MenuItem>
        ))}
      </Select>
      {order.deliveryStatus !== "sent" &&
        order.deliveryStatus !== "sending" && (
          <Button
            disabled={retry.isPending || isPending}
            onClick={() => setConfirm(true)}
          >
            Retry Slack notification
          </Button>
        )}
      <Dialog
        open={confirm}
        onClose={() => !retry.isPending && setConfirm(false)}
      >
        <DialogTitle>Retry Slack notification?</DialogTitle>
        <DialogContent>
          The order is already saved. Retrying an interrupted or unknown
          delivery can create a duplicate Slack message. This does not create
          another order.
        </DialogContent>
        <DialogActions>
          <Button disabled={retry.isPending} onClick={() => setConfirm(false)}>
            Cancel
          </Button>
          <Button disabled={retry.isPending} onClick={() => retry.mutate()}>
            Retry notification
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export function LoungeOrderList() {
  const { canAccess } = useCanAccess({
    resource: "lounge-orders",
    action: "list",
  });
  const options = useQuery({
    queryKey: ["lounge", "filter-options"],
    queryFn: () => loungeRequest<FilterOptions>("lounge-orders/filter-options"),
    enabled: Boolean(canAccess),
  });
  if (!canAccess)
    return (
      <Alert severity="info">
        You do not have permission to view lounge orders.
      </Alert>
    );
  return (
    <>
      {options.error && (
        <Alert severity="error">
          Could not load historical filters.{" "}
          <Button onClick={() => void options.refetch()}>Retry</Button>
        </Alert>
      )}
      <List
        sort={{ field: "id", order: "DESC" }}
        filterDefaultValues={{ isTest: false }}
        filters={[
          <SelectInput
            key="company"
            source="companyId"
            label="Company"
            {...emptyFilterOption}
            choices={options.data?.companies ?? []}
            alwaysOn
          />,
          <SelectInput
            key="category"
            source="categoryId"
            label="Category"
            {...emptyFilterOption}
            choices={options.data?.categories ?? []}
            alwaysOn
          />,
          <SelectInput
            key="item"
            source="itemId"
            label="Item"
            {...emptyFilterOption}
            choices={options.data?.items ?? []}
            alwaysOn
          />,
          <SelectInput
            key="status"
            source="status"
            label="Status"
            {...emptyFilterOption}
            optionText={(choice: { id: string }) => (
              <OrderStatus value={choice.id} />
            )}
            choices={statuses}
            alwaysOn
          />,
          <SelectInput
            key="test"
            source="isTest"
            label="Environment"
            {...emptyFilterOption}
            choices={[
              { id: false, name: "Real orders" },
              { id: true, name: "Test orders" },
            ]}
            alwaysOn
          />,
        ]}
      >
        <OrderTotals />
        <Datagrid rowClick="show" bulkActionButtons={false}>
          <NumberField source="id" label="Reference" />
          <TextField source="companyName" label="Company" />
          <DateField source="createdAt" showTime />
          <OrderStatusField source="status" />
          <BooleanField source="isTest" label="Test" />
          <OrderStatusField
            source="deliveryStatus"
            label="Slack notification"
          />
          <ShowButton />
        </Datagrid>
      </List>
    </>
  );
}

export function LoungeOrderShow() {
  return (
    <Show actions={<LoungeOrderShowActions />}>
      <LoungeOrderDetails />
    </Show>
  );
}

export function LoungeOrderShowActions() {
  return (
    <TopToolbar>
      <ListButton label="Back to all orders" />
    </TopToolbar>
  );
}

export function LoungeOrderDetails() {
  return (
    <Box
      sx={{
        display: "grid",
        gridTemplateColumns: {
          xs: "minmax(0, 1fr)",
          md: "320px minmax(0, 1fr)",
        },
        alignItems: "start",
        gap: 3,
        p: 2,
      }}
    >
      <SimpleShowLayout sx={{ p: 0, minWidth: 0 }}>
        <NumberField source="id" label="Reference" />
        <TextField source="companyName" label="Company at submission" />
        <DateField source="createdAt" showTime />
        <BooleanField source="isTest" label="Test order" />
        <OrderStatusField source="status" />
        <OrderStatusField source="deliveryStatus" label="Slack notification" />
        <TextField source="deliveryError" label="Notification error" />
        <LoungeOrderActions />
      </SimpleShowLayout>
      <Box
        component="section"
        aria-labelledby="lounge-order-breakdown"
        sx={{
          minWidth: 0,
          border: 1,
          borderColor: "divider",
          borderRadius: 2,
          overflow: "hidden",
        }}
      >
        <Box sx={{ px: 2, py: 1.5, bgcolor: "action.selected" }}>
          <Typography
            id="lounge-order-breakdown"
            component="h2"
            variant="h6"
            sx={{ fontWeight: 700 }}
          >
            Order breakdown
          </Typography>
        </Box>
        <Box sx={{ overflowX: "auto" }}>
          <ArrayField source="lines">
            <Datagrid
              bulkActionButtons={false}
              sx={{
                "& .RaDatagrid-headerCell": { fontWeight: 700 },
                "& .RaDatagrid-rowCell:last-child": { fontWeight: 700 },
              }}
            >
              <TextField
                source="categoryName"
                label="Category"
                sortable={false}
              />
              <TextField source="itemName" label="Item" sortable={false} />
              <NumberField
                source="quantity"
                sortable={false}
                sx={{ fontWeight: 700 }}
              />
            </Datagrid>
          </ArrayField>
        </Box>
      </Box>
    </Box>
  );
}
