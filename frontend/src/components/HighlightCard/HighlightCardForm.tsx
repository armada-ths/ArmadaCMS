import { SimpleForm, TextInput } from "react-admin";
import { MarkdownInput } from "../shared/MarkdownInput";

export const HighlightCardForm = () => (
  <SimpleForm>
    <TextInput
      source="title"
      label="Title"
      helperText="Main heading for the highlight card"
      fullWidth
    />

    <TextInput
      source="titleSv"
      label="Title (Swedish)"
      helperText="Swedish translation of the main heading (optional)"
      fullWidth
    />

    <TextInput
      source="subtitle"
      label="Subtitle"
      helperText="Secondary heading text"
      fullWidth
    />

    <TextInput
      source="subtitleSv"
      label="Subtitle (Swedish)"
      helperText="Swedish translation of the secondary heading (optional)"
      fullWidth
    />

    <MarkdownInput
      source="description"
      label="Description (Markdown)"
      enableImageFeatures={false}
    />

    <MarkdownInput
      source="descriptionSv"
      label="Description (Swedish, Markdown)"
      enableImageFeatures={false}
    />

    <TextInput
      source="brand"
      label="Brand"
      helperText='Brand name (defaults to "ARMADA" if empty)'
      fullWidth
    />

    <TextInput
      source="linkText"
      label="Link Text"
      helperText="Text to display for the link in the subtitle (leave empty for no link)"
      fullWidth
    />

    <TextInput source="linkTextSv" label="Link text (Swedish)" fullWidth />

    <TextInput
      source="linkUrl"
      label="Link URL"
      helperText="URL for the link in the subtitle (leave empty for no link)"
      fullWidth
    />

    <TextInput
      source="ctaEventName"
      label="CTA Event Name"
      helperText="Event name for Vercel Analytics tracking (e.g. student_signup_click)"
      fullWidth
    />
  </SimpleForm>
);
