import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
  type Dispatch,
  type SetStateAction,
} from "react";
import type { TranslationRecord } from "./api";
export interface TranslationDraft {
  record: TranslationRecord;
  texts: Record<string, string>;
  selected: string[];
}
type Drafts = Record<string, TranslationDraft>;
const DraftContext = createContext<{
  drafts: Drafts;
  setDrafts: Dispatch<SetStateAction<Drafts>>;
} | null>(null);
export function TranslationDraftProvider({
  children,
}: {
  children: ReactNode;
}) {
  const [drafts, setDrafts] = useState<Drafts>({});
  const dirty = Object.values(drafts).some(
    (d) => Object.keys(d.texts).length > 0,
  );
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  return (
    <DraftContext.Provider value={{ drafts, setDrafts }}>
      {children}
    </DraftContext.Provider>
  );
}
export function useTranslationDrafts() {
  const value = useContext(DraftContext);
  if (!value) throw new Error("TranslationDraftProvider is missing");
  return value;
}
