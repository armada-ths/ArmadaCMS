import { Edit, EditProps, ListButton, TopToolbar } from "react-admin";
import { useQueryClient } from "@tanstack/react-query";
import { LoungeItemForm } from "./LoungeItemForm";
export const LoungeItemEdit = (props: EditProps) => {
  const client = useQueryClient();
  return (
    <Edit
      {...props}
      title="Edit lounge item"
      actions={
        <TopToolbar>
          <ListButton label="Back to catalogue" />
        </TopToolbar>
      }
      mutationMode="pessimistic"
      redirect="list"
      mutationOptions={{
        onSettled: (_data, error) => {
          if (!error)
            void client.invalidateQueries({
              queryKey: ["lounge", "catalogue"],
            });
        },
      }}
    >
      <LoungeItemForm />
    </Edit>
  );
};
