export type Section = "welcome" | "sandbox" | "kubernetes" | "app" | "marketplace";

export const SECTION_HEADER: Record<Section, string> = {
  welcome: "Welcome",
  sandbox: "Sandbox (beta)",
  kubernetes: "Kubernetes",
  app: "App",
  marketplace: "Marketplace",
};
