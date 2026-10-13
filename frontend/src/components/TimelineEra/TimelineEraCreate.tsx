import { Create, CreateProps } from "react-admin";
import { TimelineEraForm } from "./TimelineEraForm";

export const TimelineEraCreate = (props: CreateProps) => (
  <Create {...props}>
    <TimelineEraForm />
  </Create>
);
