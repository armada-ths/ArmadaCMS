import { Create, CreateProps, ListButton, TopToolbar } from "react-admin";
import { useQueryClient } from "@tanstack/react-query";
import { LoungeItemForm } from "./LoungeItemForm";
export const LoungeItemCreate = (props: CreateProps) => {
  const client = useQueryClient();
  return (
    <Create
      {...props}
      title="Add lounge item"
      actions={
        <TopToolbar>
          <ListButton label="Back to catalogue" />
        </TopToolbar>
      }
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
    </Create>
  );
};
