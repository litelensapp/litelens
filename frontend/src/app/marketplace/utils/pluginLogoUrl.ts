// The plugin logo is shipped as a real file inside the plugin's install
// directory (alongside dist/ and the binary), so it's only servable once the
// plugin is actually installed — this is the same /api/plugins/{pluginId}/*
// route the plugin's own frontend bundle loads through (see loadPluginModule.ts).
// Returns undefined when the plugin has no logo, or hasn't been installed yet;
// PluginLogo already falls back to a placeholder icon in either case.
export const pluginLogoUrl = (pluginId: string, logoFile?: string): string | undefined =>
  logoFile ? `/api/plugins/${pluginId}/${logoFile}` : undefined;
