import { Edit, EditProps } from "react-admin";
import { TimelineEntryForm } from "./TimelineEntryForm";

export const TimelineEntryEdit = (props: EditProps) => {
  return (
    <Edit {...props}>
      <TimelineEntryForm />
    </Edit>
  );
};
