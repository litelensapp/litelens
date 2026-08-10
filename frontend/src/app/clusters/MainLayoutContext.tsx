import { createContext, FC, ReactNode, use, useMemo } from "react";
import { DetailDrawerProvider } from "./shared/components/details/DetailDrawerContext";
import { UnifiedTrayProvider } from "./shared/components/trays/unified/UnifiedTrayContext";

interface MainLayoutContextValue {
  activeContext: string;
  namespace: string;
  onNamespaceChange: (ns: string) => void;
}

const MainLayoutCtx = createContext<MainLayoutContextValue | null>(null);

export const useMainLayoutContext = (): MainLayoutContextValue => {
  const ctx = use(MainLayoutCtx);
  if (!ctx) throw new Error("useMainLayoutContext must be used inside MainLayoutProvider");
  return ctx;
};

interface MainLayoutProviderProps {
  children: ReactNode;
  className?: string;
  activeContext: string;
  namespace: string;
  onNamespaceChange: (ns: string) => void;
}

export const MainLayoutProvider: FC<MainLayoutProviderProps> = ({
  children,
  className,
  activeContext,
  namespace,
  onNamespaceChange,
}) => {
  const ctxValue = useMemo<MainLayoutContextValue>(
    () => ({
      activeContext,
      namespace,
      onNamespaceChange,
    }),
    [activeContext, namespace, onNamespaceChange]
  );

  return (
    <MainLayoutCtx.Provider value={ctxValue}>
      <DetailDrawerProvider>
        <UnifiedTrayProvider>
          <div className={className}>{children}</div>
        </UnifiedTrayProvider>
      </DetailDrawerProvider>
    </MainLayoutCtx.Provider>
  );
};
