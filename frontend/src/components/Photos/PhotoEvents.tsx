import {
  Create,
  Datagrid,
  DateField,
  DateTimeInput,
  DeleteButton,
  Edit,
  List,
  NumberInput,
  required,
  SaveButton,
  SimpleForm,
  TextField,
  TextInput,
  Toolbar,
  useNotify,
  useRecordContext,
  useRefresh,
} from "react-admin";
import { useState } from "react";
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Typography,
} from "@mui/material";
import { Link } from "react-router-dom";
import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";
import { toLocalInputValue, toUTCISOString } from "../../utils/dateTimeHelpers";

type PhotoEventDates = {
  uploads_open_at?: string;
  uploads_close_at?: string;
};

const validateUploadsClose = (value: string, values: PhotoEventDates) =>
  value &&
  values.uploads_open_at &&
  new Date(value).getTime() <= new Date(values.uploads_open_at).getTime()
    ? "Uploads must close after they open"
    : undefined;

const validateGalleryClose = (value: string, values: PhotoEventDates) =>
  value &&
  values.uploads_close_at &&
  new Date(value).getTime() < new Date(values.uploads_close_at).getTime()
    ? "Gallery must close no earlier than uploads close"
    : undefined;

type PhotoEvent = {
  id: number;
  name: string;
  deletion_requested_at: string | null;
  deletion_completed_at: string | null;
};

function EventActions() {
  const record = useRecordContext<PhotoEvent>();
  const notify = useNotify();
  const refresh = useRefresh();
  const [busy, setBusy] = useState(false);
  const [confirmation, setConfirmation] = useState<"rotate" | "delete" | null>(
    null,
  );
  if (!record) return null;

  const copyLink = async (rotate = false) => {
    setBusy(true);
    try {
      const response = await httpClient(
        `${globalApi()}/photoevents/${record.id}/${rotate ? "rotate" : "link"}`,
        { method: rotate ? "POST" : "GET" },
      );
      await navigator.clipboard.writeText(
        (response.json as { url: string }).url,
      );
      notify(rotate ? "Link rotated and copied" : "Link copied", {
        type: "success",
      });
    } catch {
      notify("Could not copy the link", { type: "error" });
    } finally {
      setBusy(false);
    }
  };

  const downloadQR = async (format: "svg" | "png") => {
    const response = await fetch(
      `${globalApi()}/photoevents/${record.id}/qr?format=${format}`,
      {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("accessToken") ?? ""}`,
        },
      },
    );
    if (!response.ok) {
      notify("Could not download the QR code", { type: "error" });
      return;
    }
    const objectURL = URL.createObjectURL(await response.blob());
    const link = document.createElement("a");
    link.href = objectURL;
    link.download = `armada-${record.id}-qr.${format}`;
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
  };

  const deleteEventData = async () => {
    setBusy(true);
    try {
      await httpClient(
        `${globalApi()}/photoevents/${record.id}/retention-delete`,
        {
          method: "POST",
          body: JSON.stringify({ external_copies_handled: true }),
        },
      );
      notify("Event access disabled and deletion queued", { type: "success" });
      refresh();
    } catch {
      notify("Could not queue deletion", { type: "error" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Box
      onClick={(event) => event.stopPropagation()}
      sx={{
        display: "flex",
        alignItems: "center",
        flexWrap: "wrap",
        gap: 1,
        py: 1,
      }}
    >
      <Button
        size="small"
        variant="outlined"
        disabled={busy}
        onClick={() => void copyLink()}
      >
        Copy link
      </Button>
      <Button
        size="small"
        variant="outlined"
        disabled={busy}
        onClick={() => setConfirmation("rotate")}
      >
        Rotate
      </Button>
      <Button
        size="small"
        variant="outlined"
        onClick={() => void downloadQR("svg")}
      >
        QR SVG
      </Button>
      <Button
        size="small"
        variant="outlined"
        onClick={() => void downloadQR("png")}
      >
        QR PNG
      </Button>
      <Button
        size="small"
        variant="outlined"
        component={Link}
        to={`/admin/photoevents/${record.id}/photos`}
      >
        Manage
      </Button>
      {!record.deletion_requested_at && (
        <Button
          size="small"
          variant="outlined"
          color="error"
          disabled={busy}
          onClick={() => setConfirmation("delete")}
        >
          Request deletion
        </Button>
      )}
      <Dialog
        open={confirmation === "rotate"}
        onClose={() => setConfirmation(null)}
      >
        <DialogTitle>Rotate event link?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            The previous QR link will stop working.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmation(null)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              setConfirmation(null);
              void copyLink(true);
            }}
          >
            Rotate link
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={confirmation === "delete"}
        onClose={() => setConfirmation(null)}
      >
        <DialogTitle>Delete event data?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This disables guest access and queues permanent deletion of the
            event photos and exports. Confirm that THS-controlled marketing
            copies and posts have been handled manually.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmation(null)}>Cancel</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              setConfirmation(null);
              void deleteEventData();
            }}
          >
            Request deletion
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

function PhotoEventToolbar() {
  const record = useRecordContext<PhotoEvent>();
  return (
    <Toolbar>
      {record?.deletion_requested_at ? (
        <Typography variant="body2" color="text.secondary">
          Event deletion has been requested.
        </Typography>
      ) : (
        <SaveButton />
      )}
      {record?.deletion_completed_at && <DeleteButton />}
    </Toolbar>
  );
}

const EventForm = ({ editing = false }: { editing?: boolean }) => (
  <SimpleForm toolbar={editing ? <PhotoEventToolbar /> : undefined}>
    <TextInput
      source="name"
      label="Event name"
      validate={required()}
      fullWidth
    />
    <TextInput source="description" label="Description" multiline fullWidth />
    <DateTimeInput
      source="uploads_open_at"
      label="Uploads open"
      parse={toUTCISOString}
      format={toLocalInputValue}
      validate={required()}
    />
    <DateTimeInput
      source="uploads_close_at"
      label="Uploads close"
      parse={toUTCISOString}
      format={toLocalInputValue}
      validate={[required(), validateUploadsClose]}
    />
    <DateTimeInput
      source="gallery_close_at"
      label="Gallery closes"
      parse={toUTCISOString}
      format={toLocalInputValue}
      validate={[required(), validateGalleryClose]}
    />
    <NumberInput
      source="max_photos_per_guest"
      label="Photos per guest"
      min={1}
      max={25}
      defaultValue={25}
    />
  </SimpleForm>
);

export const PhotoEventList = () => (
  <List perPage={25} pagination={false}>
    <Datagrid rowClick="edit">
      <TextField source="name" label="Event" />
      <DateField source="uploads_close_at" label="Uploads close" showTime />
      <EventActions />
    </Datagrid>
  </List>
);
export const PhotoEventCreate = () => (
  <Create>
    <EventForm />
  </Create>
);
export const PhotoEventEdit = () => (
  <Edit>
    <EventForm editing />
  </Edit>
);
