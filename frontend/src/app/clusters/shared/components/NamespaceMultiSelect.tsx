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

  const [open, setOpen] = useState(false);
  const [draftNamespaces, setDraftNamespaces] = useState<string[]>(namespaces);

  // Derive trigger label from the last confirmed selection, not the in-progress draft.
  const triggerLabel = useMemo(() => {
    if (namespaces.length === 0) {
      return "All namespaces";
    }
    if (namespaces.length === 1) {
      return namespaces[0];
    }
    return `${namespaces.length} namespaces`;
  }, [namespaces]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      // Reset the draft to the last confirmed selection each time the popover opens.
      setDraftNamespaces(namespaces);
    }
    setOpen(nextOpen);
  };

  const handleSelectAll = () => {
    setDraftNamespaces(sortedNamespaces);
  };

  const handleClearAll = () => {
    setDraftNamespaces([]);
  };

  const handleToggleNamespace = (ns: string) => {
    setDraftNamespaces((prev) => {
      const set = new Set(prev);
      if (set.has(ns)) {
        set.delete(ns);
      } else {
        set.add(ns);
      }
      return Array.from(set);
    });
  };

  const handleConfirm = () => {
    onNamespacesChange(draftNamespaces);
  };

  if (sortedNamespaces.length === 0) {
    return (
      <button
        type="button"
        disabled
        className="text-muted-foreground border-input bg-background inline-flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 text-sm"
      >
        No namespaces available
      </button>
    );
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger
        disabled={disabled}
        className={cn(
          "border-input bg-background text-foreground inline-flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 text-sm font-medium",
          "hover:bg-accent/50 focus:ring-ring focus:outline-none focus:ring-2 focus:ring-offset-2",
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
              className="hover:bg-accent/50 flex items-center gap-2 rounded px-2 py-1.5"
            >
              <Checkbox
                checked={draftNamespaces.includes(ns)}
                onCheckedChange={() => handleToggleNamespace(ns)}
                id={`ns-${ns}`}
              />
              <label
                htmlFor={`ns-${ns}`}
                className="flex-1 cursor-pointer text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                {ns}
              </label>
            </div>
          ))}
        </div>

        <Separator className="my-2" />

        <div className="flex gap-2 px-2 py-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleClearAll}
            className="flex-1 text-xs"
          >
            Clear
          </Button>
          <Button type="button" size="sm" onClick={handleConfirm} className="flex-1 text-xs">
            Confirm
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
