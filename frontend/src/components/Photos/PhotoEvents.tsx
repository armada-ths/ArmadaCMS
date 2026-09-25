import {
  BooleanInput,
  Create,
  Datagrid,
  DateField,
  DateTimeInput,
  Edit,
  List,
  NumberInput,
  required,
  SimpleForm,
  TextField,
  TextInput,
  useNotify,
  useRecordContext,
} from "react-admin";
import { useState } from "react";
import { Link } from "react-router-dom";
import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";

type PhotoEvent = { id: number; name: string; active: boolean };

function EventActions() {
  const record = useRecordContext<PhotoEvent>();
  const notify = useNotify();
  const [busy, setBusy] = useState(false);
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

  return (
    <div className="flex gap-2" onClick={(event) => event.stopPropagation()}>
      <button type="button" disabled={busy} onClick={() => void copyLink()}>
        Copy link
      </button>
      <button
        type="button"
        disabled={busy}
        onClick={() => {
          if (
            window.confirm("The previous QR link will stop working. Rotate it?")
          )
            void copyLink(true);
        }}
      >
        Rotate
      </button>
      <button type="button" onClick={() => void downloadQR("svg")}>
        QR SVG
      </button>
      <button type="button" onClick={() => void downloadQR("png")}>
        QR PNG
      </button>
      <Link to={`/photoevents/${record.id}/photos`}>Moderate</Link>
    </div>
  );
}

const EventForm = () => (
  <SimpleForm>
    <TextInput
      source="name"
      label="Event name"
      validate={required()}
      fullWidth
    />
    <TextInput source="slug" label="Slug" validate={required()} fullWidth />
    <TextInput source="description" label="Description" multiline fullWidth />
    <DateTimeInput
      source="uploads_open_at"
      label="Uploads open"
      validate={required()}
    />
    <DateTimeInput
      source="uploads_close_at"
      label="Uploads close"
      validate={required()}
    />
    <DateTimeInput
      source="gallery_close_at"
      label="Gallery closes"
      validate={required()}
    />
    <DateTimeInput
      source="delete_after"
      label="Delete all photos after"
      validate={required()}
    />
    <TextInput
      source="privacy_url"
      label="Privacy information URL (HTTPS)"
      fullWidth
    />
    <NumberInput
      source="max_photos_per_guest"
      label="Photos per guest"
      min={1}
      max={25}
      defaultValue={25}
    />
    <BooleanInput source="active" label="Active event" />
  </SimpleForm>
);

export const PhotoEventList = () => (
  <List perPage={25} pagination={false}>
    <Datagrid rowClick="edit">
      <TextField source="name" label="Event" />
      <TextField source="slug" label="Slug" />
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
    <EventForm />
  </Edit>
);
