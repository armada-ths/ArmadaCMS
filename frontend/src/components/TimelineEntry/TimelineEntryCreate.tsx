import { Create, CreateProps } from "react-admin";
import { TimelineEntryForm } from "./TimelineEntryForm";

export const TimelineEntryCreate = (props: CreateProps) => (
  <Create {...props}>
    <TimelineEntryForm />
  </Create>
);
