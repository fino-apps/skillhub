/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SKILLHUB_DESKTOP?: string;
  readonly VITE_APP_ID?: string;
  readonly VITE_API_BASE?: string;
  readonly VITE_DESKTOP_DIRECTORY_PICKER?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
