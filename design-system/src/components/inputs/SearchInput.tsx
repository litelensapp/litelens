import { FC, InputHTMLAttributes } from "react";
import { SearchIcon } from "../../atoms/icon";
import { Input } from "../../atoms/input";
import { cn } from "../../utils/common";

interface SearchInputProps extends InputHTMLAttributes<HTMLInputElement> {
  /** Width class for the wrapper (e.g., "w-68"). Defaults to "w-68" if not provided. */
  wrapperClassName?: string;
}

export const SearchInput: FC<SearchInputProps> = ({
  wrapperClassName = "w-68",
  className,
  ...props
}) => {
  return (
    <div className={cn("relative", wrapperClassName)}>
      <SearchIcon className="text-muted-foreground absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2" />
      <Input className={cn("pl-8 text-xs", className)} {...props} />
    </div>
  );
};
