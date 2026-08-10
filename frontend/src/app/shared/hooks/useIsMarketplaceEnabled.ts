export const useIsMarketplaceEnabled = (): boolean => {
  return import.meta.env.VITE_MARKETPLACE_ENABLED !== "false";
};
