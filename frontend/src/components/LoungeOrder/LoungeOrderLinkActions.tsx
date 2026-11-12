import { useState } from "react";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
} from "@mui/material";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCanAccess, useNotify } from "react-admin";
import { authenticatedFetch } from "../../context/authenticatedHttpClient";
import globalApi from "../../context/globalApi";
import { loungeRequest } from "../LoungeItem/catalogue";

const linkQueryKey = ["lounge", "order-link"];
const linkPath = "lounge-orders/link";

async function fetchQR() {
  const response = await authenticatedFetch(`${globalApi()}/${linkPath}/qr`);
  if (!response.ok) throw new Error("Could not load the order QR code");
  return response.blob();
}

export function LoungeOrderLinkActions() {
  const { canAccess } = useCanAccess({
    resource: "lounge-orders",
    action: "edit",
  });
  const notify = useNotify();
  const client = useQueryClient();
  const [confirm, setConfirm] = useState(false);
  const link = useQuery({
    queryKey: linkQueryKey,
    queryFn: () => loungeRequest<{ url: string }>(linkPath),
    enabled: Boolean(canAccess),
  });
  const rotate = useMutation({
    mutationFn: () =>
      loungeRequest<{ url: string }>(`${linkPath}/rotate`, "POST"),
    onSuccess: (result) => {
      client.setQueryData(linkQueryKey, result);
      notify("Order link updated", { type: "success" });
    },
    onError: () => notify("Could not update the order link", { type: "error" }),
    onSettled: () => client.invalidateQueries({ queryKey: linkQueryKey }),
  });
  if (!canAccess) return null;
  if (link.isError)
    return (
      <Alert severity="error">
        Could not load the order link.{" "}
        <Button onClick={() => void link.refetch()}>Retry</Button>
      </Alert>
    );

  const url = link.data?.url;
  const create = url === "";
  const busy = link.isPending || rotate.isPending;
  const copyLink = async () => {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      notify("Order link copied", { type: "success" });
    } catch {
      notify("Could not copy the order link", { type: "error" });
    }
  };
  const copyQR = async () => {
    try {
      await navigator.clipboard.write([
        new ClipboardItem({ "image/png": fetchQR() }),
      ]);
      notify("Order QR code copied", { type: "success" });
    } catch {
      notify("Could not copy the order QR code", { type: "error" });
    }
  };

  return (
    <>
      <Stack direction="row" sx={{ flexWrap: "wrap", gap: 1, py: 1 }}>
        <Button
          variant="outlined"
          disabled={!url || busy}
          onClick={() => void copyLink()}
        >
          Copy order link
        </Button>
        <Button
          variant="outlined"
          disabled={!url || busy}
          onClick={() => void copyQR()}
        >
          Copy QR code
        </Button>
        <Button
          variant="outlined"
          disabled={busy}
          onClick={() => setConfirm(true)}
        >
          {create ? "Create order link" : "Rotate order link"}
        </Button>
      </Stack>
      <Dialog open={confirm} onClose={() => setConfirm(false)}>
        <DialogTitle>
          {create ? "Create order link?" : "Rotate order link?"}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            Previous order links and QR codes will stop working. Share the new
            link with exhibitors.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirm(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              setConfirm(false);
              rotate.mutate();
            }}
          >
            {create ? "Create link" : "Rotate link"}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
