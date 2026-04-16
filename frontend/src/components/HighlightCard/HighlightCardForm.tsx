import { SimpleForm, TextInput } from "react-admin";

export const HighlightCardForm = () => (
  <SimpleForm>
    <TextInput
      source="title"
      label="Title"
      helperText="Main heading for the highlight card"
      fullWidth
    />

    <TextInput
      source="subtitle"
      label="Subtitle"
      helperText="Secondary heading text"
      fullWidth
    />

    <TextInput
      source="description"
      label="Description"
      helperText="Main content/body text of the card"
      multiline
      rows={4}
      fullWidth
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
