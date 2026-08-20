import { Button } from "../atoms/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "../atoms/tooltip";
import { cn } from "../utils/common";
import { FC, MouseEvent, ReactNode, useLayoutEffect, useRef, useState } from "react";

interface ResourceLinkProps {
  children: ReactNode;
  className?: string;
  truncate?: boolean;
  truncateTextClassName?: string;
  onClick?: (e: MouseEvent) => void;
}

export const ResourceLink: FC<ResourceLinkProps> = ({
  children,
  className,
  truncate,
  truncateTextClassName,
  onClick,
}) => {
  const ref = useRef<HTMLSpanElement>(null);
  const [isOverflowing, setIsOverflowing] = useState(false);

  useLayoutEffect(() => {
    if (!truncate) return;
    const el = ref.current;
    if (el) setIsOverflowing(el.scrollWidth > el.clientWidth);
  }, [truncate, children]);

  const inner = onClick ? (
    <Button
      variant="link"
      type="button"
      size="xs"
      className={cn("text-info h-auto w-fit p-0", className)}
      onClick={onClick}
    >
      {truncate ? (
        <span
          ref={ref}
          className={cn(
            "max-w-65 inline-block truncate group-hover/button:[box-shadow:0_1px_0_0_currentColor]",
            truncateTextClassName
          )}
        >
          {children}
        </span>
      ) : (
        children
      )}
    </Button>
  ) : (
    <span className={cn("text-info cursor-default", className)}>
      {truncate ? (
        <span ref={ref} className={cn("max-w-65 inline-block truncate", truncateTextClassName)}>
          {children}
        </span>
      ) : (
        children
      )}
    </span>
  );

  if (!truncate || !isOverflowing) return inner;

  return (
    <Tooltip>
      <TooltipTrigger render={inner} />
      <TooltipContent>
        <span className="break-all font-mono">{children}</span>
      </TooltipContent>
    </Tooltip>
  );
};
