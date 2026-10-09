import { Edit, EditProps } from "react-admin";
import { TimelineEraForm } from "./TimelineEraForm";

export const TimelineEraEdit = (props: EditProps) => (
  <Edit {...props}>
    <TimelineEraForm />
  </Edit>
);
