import {
  Button,
  RaRecord,
  useCanAccess,
  useNotify,
  useRecordContext,
  useRefresh,
  useUpdate,
} from "react-admin";
import Block from "@mui/icons-material/Block";
import CheckCircle from "@mui/icons-material/CheckCircle";
import { useQueryClient } from "@tanstack/react-query";

interface LoungeItemRecord extends RaRecord {
  soldOut: boolean;
}

export const SoldOutButton = () => {
  const record = useRecordContext<LoungeItemRecord>();
  const { canAccess } = useCanAccess({
    resource: "lounge-items",
    action: "edit",
  });
  const [update, { isPending }] = useUpdate();
  const notify = useNotify();
  const refresh = useRefresh();
  const client = useQueryClient();
  if (!record || !canAccess) return null;
  const label = record.soldOut ? "Mark available" : "Mark sold out";
  return (
    <Button
      label={label}
      aria-label={label}
      disabled={isPending}
      onClick={(event) => {
        event.stopPropagation();
        update(
          "lounge-items",
          {
            id: record.id,
            data: { soldOut: !record.soldOut },
            previousData: record,
          },
          {
            mutationMode: "pessimistic",
            onSuccess: () => {
              notify(
                record.soldOut
                  ? "Item marked available"
                  : "Item marked sold out",
                { type: "success" },
              );
              refresh();
              void client.invalidateQueries({
                queryKey: ["lounge", "catalogue"],
              });
            },
            onError: () =>
              notify("Could not change item availability. Please try again.", {
                type: "error",
              }),
          },
        );
      }}
    >
      {record.soldOut ? <CheckCircle /> : <Block />}
    </Button>
  );
};
