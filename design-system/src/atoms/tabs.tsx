import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../utils/common";

function Tabs({ className, orientation = "horizontal", ...props }: TabsPrimitive.Root.Props) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      orientation={orientation}
      className={cn(
        "group/tabs data-horizontal:grid-rows-[auto_1fr] data-vertical:grid-cols-[auto_1fr] grid gap-2",
        className
      )}
      {...props}
    />
  );
}

const tabsListVariants = cva(
  "group/tabs-list relative inline-flex w-fit items-center justify-center rounded-lg p-[3px] text-muted-foreground group-data-horizontal/tabs:h-8 group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col data-[variant=line]:rounded-none",
  {
    variants: {
      variant: {
        default: "bg-muted",
        line: "gap-1 bg-transparent",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

interface TabsListProps extends TabsPrimitive.List.Props, VariantProps<typeof tabsListVariants> {
  /** className merged onto the auto-rendered TabsIndicator (e.g. to override its default bg-primary color) */
  indicatorClassName?: string;
}

function TabsList({
  className,
  variant = "default",
  indicatorClassName,
  children,
  ...props
}: TabsListProps) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      data-variant={variant}
      className={cn(tabsListVariants({ variant }), className)}
      {...props}
    >
      {children}
      <TabsIndicator variant={variant ?? "default"} className={indicatorClassName} />
    </TabsPrimitive.List>
  );
}

function TabsTrigger({ className, ...props }: TabsPrimitive.Tab.Props) {
  return (
    <TabsPrimitive.Tab
      data-slot="tabs-trigger"
      className={cn(
        "text-foreground/60 group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start hover:text-foreground focus-visible:border-ring focus-visible:ring-ring/75 focus-visible:outline-ring has-data-[icon=inline-end]:pr-1 has-data-[icon=inline-start]:pl-1 dark:text-muted-foreground dark:hover:text-foreground transition-interactive relative z-10 inline-flex h-[calc(100%-1px)] flex-1 cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap rounded-md border border-transparent bg-transparent px-1.5 py-0.5 text-sm font-medium focus-visible:outline-1 focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0",
        "data-active:text-foreground dark:data-active:text-foreground",
        className
      )}
      {...props}
    />
  );
}

interface TabsIndicatorProps extends TabsPrimitive.Indicator.Props {
  variant?: "default" | "line";
}

function TabsIndicator({ className, variant = "default", ...props }: TabsIndicatorProps) {
  return (
    <TabsPrimitive.Indicator
      data-slot="tabs-indicator"
      className={cn(
        "transition-interactive pointer-events-none absolute z-0",
        variant === "line"
          ? cn(
              "bg-primary",
              "group-data-horizontal/tabs:bottom-0 group-data-horizontal/tabs:left-(--active-tab-left) group-data-horizontal/tabs:h-0.5 group-data-horizontal/tabs:w-(--active-tab-width)",
              "group-data-vertical/tabs:right-0 group-data-vertical/tabs:top-(--active-tab-top) group-data-vertical/tabs:h-(--active-tab-height) group-data-vertical/tabs:w-0.5"
            )
          : "left-(--active-tab-left) top-(--active-tab-top) h-(--active-tab-height) w-(--active-tab-width) bg-background dark:border-input dark:bg-input/30 rounded-md border border-transparent shadow-sm",
        className
      )}
      {...props}
    />
  );
}

function TabsContent({ className, ...props }: TabsPrimitive.Panel.Props) {
  return (
    <TabsPrimitive.Panel
      data-slot="tabs-content"
      className={cn(
        "transition-fade data-ending-style:opacity-0 data-hidden:opacity-0 data-starting-style:opacity-0 group-data-horizontal/tabs:row-start-2 group-data-horizontal/tabs:col-start-1 group-data-vertical/tabs:col-start-2 group-data-vertical/tabs:row-start-1 min-h-0 min-w-0 flex-1 text-sm outline-none",
        className
      )}
      {...props}
    />
  );
}

export { Tabs, TabsContent, TabsList, TabsIndicator, TabsTrigger };
