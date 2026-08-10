/**
 * Injects (or updates) a <style> tag containing a plugin's precompiled CSS,
 * which the plugin bundle exports as a string (PLUGIN_STYLES) rather than
 * shipping as a separate file. Must run once the plugin's dynamic import()
 * resolves, or its Tailwind classes render unstyled.
 *
 * Idempotent per pluginId: re-calling with the same text is a no-op; a
 * changed text (reinstall/update) replaces the tag's contents.
 *
 * No-ops when cssText is empty/undefined — older plugin bundles built before
 * plugins shipped their own CSS simply have nothing to inject here.
 */
export const ensurePluginStylesheet = (pluginId: string, cssText: string | undefined) => {
  if (!cssText) return;

  const styleId = `plugin-style-${pluginId}`;
  const existing = document.getElementById(styleId) as HTMLStyleElement | null;
  if (existing) {
    if (existing.textContent !== cssText) {
      existing.textContent = cssText;
    }
    return;
  }

  const style = document.createElement("style");
  style.id = styleId;
  style.textContent = cssText;
  document.head.appendChild(style);
};
