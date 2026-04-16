import { Edit, EditProps } from "react-admin";
import { HighlightCardForm } from "./HighlightCardForm";

export const HighlightCardEdit = (props: EditProps) => (
  <Edit {...props}>
    <HighlightCardForm />
  </Edit>
);
