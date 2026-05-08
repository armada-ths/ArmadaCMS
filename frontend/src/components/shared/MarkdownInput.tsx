import { useInput, InputProps } from "react-admin";
import MDEditor, { commands, type RefMDEditor } from "@uiw/react-md-editor";
import { useCallback, useRef, useState } from "react";
import { useTheme } from "@mui/material/styles";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Stack,
} from "@mui/material";
import globalApi from "../../context/globalApi";

const endpoint = globalApi();

interface MarkdownInputProps extends InputProps {
  label?: string;
}

interface EditingImg {
  src: string;
  alt: string;
  width: number;
  height: number;
}

// Parse ![alt|WxH](src) or ![alt](src)
function parseImgAlt(raw: string): {
  alt: string;
  width: number;
  height: number;
} {
  const m = raw.match(/^(.+?)\|(\d+)x(\d+)$/);
  return m
    ? { alt: m[1], width: Number(m[2]), height: Number(m[3]) }
    : { alt: raw, width: 800, height: 450 };
}

function PreviewImage({
  src,
  alt,
  width,
  height,
  onResize,
}: {
  src: string;
  alt: string;
  width: number;
  height: number;
  onResize: (img: EditingImg) => void;
}) {
  return (
    <span
      style={{
        position: "relative",
        display: "inline-block",
        cursor: "pointer",
      }}
      title="Click to resize"
      onClick={() => onResize({ src, alt, width, height })}
    >
      <img
        src={src}
        alt={alt}
        width={width}
        height={height}
        style={{ maxWidth: "100%", display: "block", height: "auto" }}
      />
      <span
        style={{
          position: "absolute",
          top: 4,
          right: 4,
          background: "rgba(0,0,0,0.55)",
          color: "#fff",
          padding: "2px 7px",
          borderRadius: 4,
          fontSize: 11,
          pointerEvents: "none",
        }}
      >
        {width}×{height} ✎
      </span>
    </span>
  );
}

export const MarkdownInput = ({ label, ...props }: MarkdownInputProps) => {
  const {
    field: { value, onChange },
  } = useInput(props);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const editorRef = useRef<RefMDEditor | null>(null);

  const muiTheme = useTheme();
  const colorMode = muiTheme.palette.mode;

  const [editingImg, setEditingImg] = useState<EditingImg | null>(null);
  const [editWidth, setEditWidth] = useState("");
  const [editHeight, setEditHeight] = useState("");
  const [naturalRatio, setNaturalRatio] = useState<number | null>(null);

  const openImgResize = useCallback((img: EditingImg) => {
    setEditingImg(img);
    setEditWidth(String(img.width));
    setEditHeight(String(img.height));
    setNaturalRatio(null);
    const el = new window.Image();
    el.onload = () => {
      if (el.naturalWidth > 0 && el.naturalHeight > 0) {
        const ratio = el.naturalWidth / el.naturalHeight;
        setNaturalRatio(ratio);
      }
    };
    el.src = img.src;
  }, []);

  const applyImgResize = useCallback(() => {
    if (!editingImg) return;
    const { src, alt } = editingImg;
    const w = Number(editWidth) || editingImg.width;
    const h = Number(editHeight) || editingImg.height;
    const escapedSrc = src.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const updated = (value || "").replace(
      new RegExp(`!\\[[^\\]]*\\]\\(${escapedSrc}\\)`),
      `![${alt}|${w}x${h}](${src})`,
    );
    onChange(updated);
    setEditingImg(null);
  }, [editingImg, editWidth, editHeight, value, onChange]);

  const uploadImage = useCallback(
    async (file: File): Promise<string | null> => {
      const token = localStorage.getItem("accessToken") || "";
      const formData = new FormData();
      formData.append("file", file);

      try {
        const res = await fetch(`${endpoint}/blogposts/upload`, {
          method: "POST",
          headers: { Authorization: `Bearer ${token}` },
          body: formData,
        });

        if (!res.ok) {
          const text = await res.text();
          alert(`Image upload failed: ${text}`);
          return null;
        }

        const data = await res.json();
        return data.url as string;
      } catch {
        alert("Image upload failed");
        return null;
      }
    },
    [],
  );

  const handleImageUpload = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleFileChange = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;

      const url = await uploadImage(file);
      if (url) {
        // Load the image to get its actual pixel dimensions
        const { w, h } = await new Promise<{ w: number; h: number }>(
          (resolve) => {
            const img = new window.Image();
            img.onload = () =>
              resolve({
                w: img.naturalWidth || 800,
                h: img.naturalHeight || 450,
              });
            img.onerror = () => resolve({ w: 800, h: 450 });
            img.src = url;
          },
        );
        const markdown = `![${file.name}|${w}x${h}](${url})`;
        const textarea = editorRef.current?.textarea;
        if (textarea) {
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          const currentValue = value || "";
          const newValue =
            currentValue.substring(0, start) +
            markdown +
            currentValue.substring(end);
          onChange(newValue);
        } else {
          onChange((value || "") + "\n" + markdown);
        }
      }

      // Reset file input so same file can be selected again
      e.target.value = "";
    },
    [uploadImage, onChange, value],
  );

  // Custom image command that triggers file upload
  const imageUploadCommand: commands.ICommand = {
    name: "image-upload",
    keyCommand: "image-upload",
    buttonProps: { "aria-label": "Upload image", title: "Upload image" },
    icon: (
      <svg viewBox="0 0 20 20" width="12" height="12" fill="currentColor">
        <path d="M15 9c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm4-7H1c-.55 0-1 .45-1 1v14c0 .55.45 1 1 1h18c.55 0 1-.45 1-1V3c0-.55-.45-1-1-1zm-1 13l-6-5-2 2-4-5-4 8V4h16v11z" />
      </svg>
    ),
    execute: () => {
      handleImageUpload();
    },
  };

  return (
    <div
      style={{ marginBottom: "1em", width: "100%" }}
      data-color-mode={colorMode}
    >
      {label && (
        <label
          style={{
            display: "block",
            marginBottom: "0.5em",
            fontWeight: 500,
            fontSize: "0.85rem",
            color: muiTheme.palette.text.secondary,
          }}
        >
          {label}
        </label>
      )}
      <MDEditor
        ref={editorRef}
        value={value || ""}
        onChange={(val) => onChange(val || "")}
        height={400}
        previewOptions={{
          components: {
            img: (props) => {
              const src = typeof props.src === "string" ? props.src : "";
              const { alt, width, height } = parseImgAlt(props.alt ?? "");
              return (
                <PreviewImage
                  src={src}
                  alt={alt}
                  width={width}
                  height={height}
                  onResize={openImgResize}
                />
              );
            },
          },
        }}
        commands={[
          commands.bold,
          commands.italic,
          commands.strikethrough,
          commands.hr,
          commands.divider,
          commands.heading1,
          commands.heading2,
          commands.heading3,
          commands.divider,
          commands.unorderedListCommand,
          commands.orderedListCommand,
          commands.checkedListCommand,
          commands.divider,
          commands.link,
          commands.quote,
          commands.code,
          commands.divider,
          imageUploadCommand,
        ]}
        extraCommands={[commands.fullscreen]}
      />
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp,image/gif"
        style={{ display: "none" }}
        onChange={handleFileChange}
      />
      <p
        style={{
          fontSize: "0.75rem",
          color: muiTheme.palette.text.secondary,
          marginTop: "0.25em",
        }}
      >
        Supports Markdown. Use the image button to upload and insert inline
        images.
      </p>

      <Dialog open={!!editingImg} onClose={() => setEditingImg(null)}>
        <DialogTitle>Resize image</DialogTitle>
        <DialogContent>
          <Stack direction="row" spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Width (px)"
              type="number"
              value={editWidth}
              onChange={(e) => {
                const w = e.target.value;
                setEditWidth(w);
                if (Number(w) > 0) {
                  const ratio =
                    naturalRatio ??
                    (editingImg ? editingImg.height / editingImg.width : null);
                  if (ratio != null)
                    setEditHeight(String(Math.round(Number(w) / ratio)));
                }
              }}
              slotProps={{ htmlInput: { min: 1 } }}
              size="small"
            />
            <TextField
              label="Height (px)"
              type="number"
              value={editHeight}
              onChange={(e) => {
                const h = e.target.value;
                setEditHeight(h);
                if (Number(h) > 0) {
                  const ratio =
                    naturalRatio ??
                    (editingImg ? editingImg.width / editingImg.height : null);
                  if (ratio != null)
                    setEditWidth(String(Math.round(Number(h) * ratio)));
                }
              }}
              slotProps={{ htmlInput: { min: 1 } }}
              size="small"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditingImg(null)}>Cancel</Button>
          <Button variant="contained" onClick={applyImgResize}>
            Apply
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
};
