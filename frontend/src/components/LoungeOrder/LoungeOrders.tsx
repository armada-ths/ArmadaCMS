import { useState } from "react";
import {
  List,
  Datagrid,
  TextField,
  DateField,
  BooleanField,
  ShowButton,
  Show,
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
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Select,
  Typography,
} from "@mui/material";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
const statuses = ["pending", "completed", "cancelled"].map((id) => ({
  id,
  name: id,
}));

function OrderTotals() {
  const { filterValues } = useListContext();
  const query = useQuery({
    queryKey: ["lounge", "totals", filterValues],
    queryFn: () =>
      loungeRequest<Total[]>(
        `lounge-orders/totals?filter=${encodeURIComponent(JSON.stringify(filterValues))}`,
      ),
  });
  const categoryTotals = new Map<number, { name: string; quantity: bigint }>();
  for (const total of query.data ?? []) {
    const previous = categoryTotals.get(total.categoryId);
    categoryTotals.set(total.categoryId, {
      name: total.categoryName,
      quantity: (previous?.quantity ?? 0n) + BigInt(total.quantity),
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
      {[...categoryTotals].map(([id, total]) => (
        <Typography key={id}>
          {total.name}: {total.quantity.toString()}
        </Typography>
      ))}
      {(query.data ?? []).map((total) => (
        <Typography
          key={`${total.categoryId}-${total.itemId}-${total.itemName}`}
          variant="body2"
        >
          {total.categoryName} / {total.itemName}: {total.quantity}
        </Typography>
      ))}
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
        aria-label="Order status"
        value={order.status}
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
            {status.name}
          </MenuItem>
        ))}
      </Select>
      {order.deliveryStatus !== "sent" && (
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
            choices={options.data?.companies ?? []}
            alwaysOn
          />,
          <SelectInput
            key="category"
            source="categoryId"
            label="Category"
            choices={options.data?.categories ?? []}
            alwaysOn
          />,
          <SelectInput
            key="item"
            source="itemId"
            label="Item"
            choices={options.data?.items ?? []}
            alwaysOn
          />,
          <SelectInput
            key="status"
            source="status"
            choices={statuses}
            helperText="Cancelled orders excluded unless selected"
            alwaysOn
          />,
          <SelectInput
            key="test"
            source="isTest"
            label="Environment"
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
          <TextField source="status" />
          <BooleanField source="isTest" label="Test" />
          <TextField source="deliveryStatus" label="Slack notification" />
          <ShowButton />
        </Datagrid>
      </List>
    </>
  );
}

export function LoungeOrderShow() {
  return (
    <Show>
      <SimpleShowLayout>
        <NumberField source="id" label="Reference" />
        <TextField source="companyName" label="Company at submission" />
        <DateField source="createdAt" showTime />
        <BooleanField source="isTest" label="Test order" />
        <TextField source="status" />
        <TextField source="deliveryStatus" label="Slack notification" />
        <TextField source="deliveryError" label="Notification error" />
        <LoungeOrderActions />
        <ArrayField source="lines">
          <Datagrid bulkActionButtons={false}>
            <TextField
              source="categoryName"
              label="Category at submission"
              sortable={false}
            />
            <TextField
              source="itemName"
              label="Item at submission"
              sortable={false}
            />
            <NumberField source="quantity" sortable={false} />
          </Datagrid>
        </ArrayField>
      </SimpleShowLayout>
    </Show>
  );
}
