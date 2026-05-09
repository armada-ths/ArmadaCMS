import { useState } from "react";
import {
  MenuItem,
  ListItemIcon,
  ListItemText,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  Button,
} from "@mui/material";
import LockIcon from "@mui/icons-material/Lock";
import { usePermissions, useNotify } from "react-admin";
import axiosInstance from "../context/axiosInstance";

const hasPerm = (perms: string[], required: string) =>
  perms.some((p) => p === "*" || p === required);

export const ChangePasswordButton = () => {
  const { permissions } = usePermissions();
  const perms: string[] = Array.isArray(permissions) ? permissions : [];
  const canChange = hasPerm(perms, "customusers.changeownpassword");
  const notify = useNotify();

  const [open, setOpen] = useState(false);
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (!canChange) return null;

  const handleClose = () => {
    setOpen(false);
    setOldPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setError("");
  };

  const handleSubmit = async () => {
    setError("");

    if (!oldPassword || !newPassword) {
      setError("All fields are required.");
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("New passwords do not match.");
      return;
    }
    if (newPassword.length < 6) {
      setError("New password must be at least 6 characters.");
      return;
    }
    if (oldPassword === newPassword) {
      setError("New password must be different from the current password.");
      return;
    }

    setLoading(true);
    try {
      await axiosInstance.put("/me/password", {
        oldPassword,
        newPassword,
      });
      notify("Password changed successfully", { type: "success" });
      handleClose();
    } catch (err: unknown) {
      if (
        typeof err === "object" &&
        err !== null &&
        "response" in err &&
        typeof (err as Record<string, unknown>).response === "object"
      ) {
        const response = (err as { response: { status?: number } }).response;
        if (response.status === 403) {
          setError("Old password is incorrect.");
        } else {
          setError("Failed to change password. Please try again.");
        }
      } else {
        setError("Failed to change password. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <MenuItem onClick={() => setOpen(true)}>
        <ListItemIcon>
          <LockIcon fontSize="small" />
        </ListItemIcon>
        <ListItemText>Change password</ListItemText>
      </MenuItem>
      <Dialog open={open} onClose={handleClose} maxWidth="xs" fullWidth>
        <DialogTitle>Change Password</DialogTitle>
        <DialogContent>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          <TextField
            label="Current password"
            type="password"
            fullWidth
            margin="normal"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            autoComplete="current-password"
          />
          <TextField
            label="New password"
            type="password"
            fullWidth
            margin="normal"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            autoComplete="new-password"
          />
          <TextField
            label="Confirm new password"
            type="password"
            fullWidth
            margin="normal"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            autoComplete="new-password"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained" disabled={loading}>
            {loading ? "Saving…" : "Change password"}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
