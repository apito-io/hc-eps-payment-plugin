export {};

declare global {
  interface Window {
    React: typeof import("react");
    antd: typeof import("antd");
    env?: { VITE_PUBLIC_API_URL?: string };
    ApitoPluginSDK: {
      version: string;
      register: (config: Record<string, unknown>) => void;
      projectId: () => string;
      url: (path: string) => string;
      publicUrl?: (path: string) => string;
      saveProjectEnv: (pluginId: string, env: Record<string, string>) => Promise<void>;
      rest: (pluginId: string) => {
        get: (path: string, init?: RequestInit) => Promise<unknown>;
        post: (path: string, body?: unknown, init?: RequestInit) => Promise<unknown>;
      };
      notify: {
        success: (msg: string) => void;
        error: (msg: string) => void;
      };
    };
  }
}
