import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";
export interface Suggestion {
  id: number;
  version: number;
  draft: string;
  generated: string;
  status: string;
  targetBefore: string | null;
  model: string;
  promptVersion: string;
  glossaryVersion: string;
  createdAt: string;
}
export interface TranslationField {
  id: string;
  format: string;
  source: string;
  published: string | null;
  status: string;
  suggestion?: Suggestion;
  warnings: string[];
  olderPolicy: boolean;
}
export interface TranslationRecord {
  id: number;
  label: string;
  fingerprint: string;
  fields: TranslationField[];
  job?: {
    id: number;
    status: string;
    outdated?: boolean;
    error?: string;
    notBefore: string;
    model: string;
    promptVersion: string;
    glossaryVersion: string;
    createdAt: string;
  };
}
export interface Resource {
  id: string;
  label: string;
  aiAllowed: boolean;
  canEdit: boolean;
  missingRecords: number;
}
export interface Overview {
  resources: Resource[];
  aiEnabled: boolean;
  dailyBudget: number;
  usage: { day: string; used: number; reserved: number };
  promptVersion: string;
  glossaryVersion: string;
}
export interface TranslationPage {
  records: TranslationRecord[];
  total: number;
}
export async function translationRequest<T>(
  path: string,
  body?: unknown,
  method = body ? "POST" : "GET",
  signal?: AbortSignal,
): Promise<T> {
  const result = await httpClient(`${globalApi()}/translations${path}`, {
    method,
    signal,
    ...(body
      ? {
          body: JSON.stringify(body),
          headers: new Headers({ "Content-Type": "application/json" }),
        }
      : {}),
  });
  return result.json as T;
}
