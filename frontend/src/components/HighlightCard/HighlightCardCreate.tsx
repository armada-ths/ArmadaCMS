import { Create, CreateProps } from "react-admin";
import { HighlightCardForm } from "./HighlightCardForm";

export const HighlightCardCreate = (props: CreateProps) => (
  <Create {...props}>
    <HighlightCardForm />
  </Create>
);
