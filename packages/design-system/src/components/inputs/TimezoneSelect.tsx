import { FC, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Button } from "../../atoms/button";
import { CheckIcon, ChevronDownIcon, SearchIcon } from "../../atoms/icon";
import { Input } from "../../atoms/input";
import { cn } from "../../utils/common";

const TIMEZONES: string[] = Intl.supportedValuesOf("timeZone");

interface DropdownRect {
  top: number;
  left: number;
  width: number;
}

interface TimezoneSelectProps {
  value: string;
  onChange: (tz: string) => void;
  "aria-labelledby"?: string;
}

export const TimezoneSelect: FC<TimezoneSelectProps> = ({
  value,
  onChange,
  "aria-labelledby": ariaLabelledBy,
}) => {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [dropdownRect, setDropdownRect] = useState<DropdownRect | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const filtered = useMemo(
    () =>
      search.trim()
        ? TIMEZONES.filter((tz) => tz.toLowerCase().includes(search.toLowerCase()))
        : TIMEZONES,
    [search]
  );

  function handleOpen() {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    setDropdownRect({ top: rect.bottom + 4, left: rect.left, width: rect.width });
    setSearch("");
    setOpen(true);
    // DOM doesn't exist until after re-render, so defer focus one frame
    requestAnimationFrame(() => searchRef.current?.focus());
  }

  useEffect(() => {
    if (!open) return;
    function handlePointerDown(e: MouseEvent) {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target) || dropdownRef.current?.contains(target)) return;
      setOpen(false);
    }
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [open]);

  return (
    <>
      <Button
        ref={triggerRef}
        variant="outline"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-labelledby={ariaLabelledBy}
        onClick={handleOpen}
        className="w-full justify-between font-normal"
      >
        <span className="truncate">{value}</span>
        <ChevronDownIcon className="text-muted-foreground size-4 shrink-0" />
      </Button>

      {open &&
        dropdownRect &&
        createPortal(
          <div
            ref={dropdownRef}
            style={{
              position: "fixed",
              top: dropdownRect.top,
              left: dropdownRect.left,
              width: dropdownRect.width,
              zIndex: 50,
            }}
            className="bg-popover text-popover-foreground ring-foreground/10 flex flex-col overflow-hidden rounded-lg shadow-md ring-1"
          >
            <div className="border-b p-1.5">
              <div className="relative">
                <SearchIcon className="text-muted-foreground pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2" />
                <Input
                  ref={searchRef}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search timezone…"
                  className="pl-8 text-sm"
                />
              </div>
            </div>
            {/* Custom searchable combobox (icons, click handlers, filtering) — no native element
                supports this, so role="listbox"/"option" is the correct WAI-ARIA pattern here,
                not a candidate for <datalist>/<option>. */}
            <div role="listbox" aria-label="Timezone" className="max-h-56 overflow-y-auto p-1">
              {filtered.length === 0 ? (
                <p className="text-muted-foreground px-3 py-4 text-center text-sm">
                  No timezone found.
                </p>
              ) : (
                filtered.map((tz) => (
                  <div
                    key={tz}
                    role="option"
                    aria-selected={tz === value}
                    tabIndex={-1}
                    onClick={() => {
                      onChange(tz);
                      setOpen(false);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        onChange(tz);
                        setOpen(false);
                      }
                    }}
                    className={cn(
                      "flex cursor-pointer items-center justify-between rounded-md px-3 py-1.5 text-sm",
                      tz === value
                        ? "bg-accent text-accent-foreground"
                        : "hover:bg-accent hover:text-accent-foreground"
                    )}
                  >
                    {tz}
                    {tz === value && <CheckIcon className="size-3.5 shrink-0" />}
                  </div>
                ))
              )}
            </div>
          </div>,
          document.body
        )}
    </>
  );
};
