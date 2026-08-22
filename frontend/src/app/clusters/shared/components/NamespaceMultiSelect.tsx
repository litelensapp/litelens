import {
  Button,
  Checkbox,
  ChevronDownIcon,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  Separator,
  cn,
} from "@litelens/design-system";
import { FC, useMemo, useState } from "react";

interface NamespaceMultiSelectProps {
  namespaces: string[];
  availableNamespaces: string[];
  onNamespacesChange: (namespaces: string[]) => void;
  disabled?: boolean;
}

export const NamespaceMultiSelect: FC<NamespaceMultiSelectProps> = ({
  namespaces,
  availableNamespaces,
  onNamespacesChange,
  disabled = false,
}) => {
  const sortedNamespaces = useMemo(
    () => availableNamespaces.slice().sort((a, b) => a.localeCompare(b)),
    [availableNamespaces]
  );

  const selectedNamespaces = useMemo(() => new Set(namespaces), [namespaces]);

  const [open, setOpen] = useState(false);

  const triggerLabel = useMemo(() => {
    if (namespaces.length === 0) {
      return "All namespaces";
    }
    if (namespaces.length === 1) {
      return namespaces[0];
    }
    return `${namespaces.length} namespaces`;
  }, [namespaces]);

  const handleSelectAll = () => {
    onNamespacesChange(sortedNamespaces);
  };

  const handleClearAll = () => {
    onNamespacesChange([]);
  };

  const handleToggleNamespace = (ns: string) => {
    const set = new Set(namespaces);
    if (set.has(ns)) {
      set.delete(ns);
    } else {
      set.add(ns);
    }
    onNamespacesChange(Array.from(set));
  };

  if (sortedNamespaces.length === 0) {
    return (
      <button
        type="button"
        disabled
        className="inline-flex items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm text-muted-foreground"
      >
        No namespaces available
      </button>
    );
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        disabled={disabled}
        className={cn(
          "inline-flex items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium text-foreground",
          "hover:bg-accent/50 focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-none",
          disabled && "cursor-not-allowed opacity-50"
        )}
        aria-label="Namespace filter"
      >
        <span className="truncate">{triggerLabel}</span>
        <ChevronDownIcon className="h-4 w-4 shrink-0" />
      </DropdownMenuTrigger>

      <DropdownMenuContent className="w-72" align="end">
        <div className="px-2 py-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleSelectAll}
            className="w-full justify-start text-xs"
          >
            Select All
          </Button>
        </div>

        <Separator />

        <div className="max-h-64 space-y-2 overflow-y-auto p-2">
          {sortedNamespaces.map((ns) => (
            <div
              key={ns}
              className="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-accent/50"
            >
              <Checkbox
                checked={selectedNamespaces.has(ns)}
                onCheckedChange={() => handleToggleNamespace(ns)}
                id={`ns-${ns}`}
              />
              <label
                htmlFor={`ns-${ns}`}
                className="flex-1 cursor-pointer text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                {ns}
              </label>
            </div>
          ))}
        </div>

        <Separator className="my-2" />

        <div className="px-2 py-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleClearAll}
            className="w-full justify-center text-xs"
          >
            Clear
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
