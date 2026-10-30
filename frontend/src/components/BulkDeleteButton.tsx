import {
  BulkDeleteButton as RaBulkDeleteButton,
  useListContext,
  useNotify,
  useRefresh,
} from "react-admin";
import { BulkDeleteError } from "../dataProvider";

/** Wait for the server before changing the list or reporting success. */
export const BulkDeleteButton = () => {
  const { onSelect } = useListContext();
  const notify = useNotify();
  const refresh = useRefresh();
  return (
    <RaBulkDeleteButton
      mutationMode="pessimistic"
      mutationOptions={{
        onError: (error: unknown) => {
          if (error instanceof BulkDeleteError) {
            onSelect(error.failedIds);
          }
          notify(
            error instanceof Error
              ? error.message
              : "ra.notification.http_error",
            {
              type: "error",
            },
          );
          refresh();
        },
      }}
    />
  );
};
