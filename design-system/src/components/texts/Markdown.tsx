import { cn } from "../../utils/common";
import { FC, JSX } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type MarkdownComponents = Record<string, (props: any) => JSX.Element>;

const markdownComponents: MarkdownComponents = {
  h1: ({ children }) => (
    <h1 className="text-foreground mb-3 mt-4 text-lg font-bold leading-tight">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-foreground border-border/50 mb-2.5 mt-3.5 border-b pb-2 text-base font-semibold leading-snug">
      {children}
    </h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-foreground mb-2 mt-3 text-sm font-semibold leading-snug">{children}</h3>
  ),
  h4: ({ children }) => (
    <h4 className="text-foreground mb-1.5 mt-2.5 text-sm font-semibold leading-snug">{children}</h4>
  ),
  h5: ({ children }) => (
    <h5 className="text-foreground mb-1.5 mt-2.5 text-xs font-semibold leading-snug">{children}</h5>
  ),
  h6: ({ children }) => (
    <h6 className="text-muted-foreground mb-1.5 mt-2.5 text-xs font-semibold leading-snug">
      {children}
    </h6>
  ),
  p: ({ children }) => <p className="text-foreground mb-2.5 text-xs leading-relaxed">{children}</p>,
  ul: ({ children }) => (
    <ul className="text-foreground mb-2.5 list-inside list-disc space-y-1 text-xs leading-relaxed">
      {children}
    </ul>
  ),
  ol: ({ children }) => (
    <ol className="text-foreground mb-2.5 list-inside list-decimal space-y-1 text-xs leading-relaxed">
      {children}
    </ol>
  ),
  li: ({ children }) => <li className="text-foreground text-xs leading-relaxed">{children}</li>,
  pre: ({ children }) => (
    <pre className="bg-muted/40 border-border/40 my-3 overflow-x-auto rounded border p-3">
      {children}
    </pre>
  ),
  code: ({ children, className }) => {
    const isBlock = !!className;
    if (isBlock) {
      return (
        <code className="text-foreground block font-mono text-xs leading-relaxed">{children}</code>
      );
    }
    return (
      <code className="bg-muted/60 text-foreground rounded px-1.5 py-0.5 font-mono text-xs">
        {children}
      </code>
    );
  },
  table: ({ children }) => (
    <div className="my-3 overflow-x-auto">
      <table className="border-collapse text-xs">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead className="bg-muted/50">{children}</thead>,
  th: ({ children }) => (
    <th className="text-foreground border-border/40 border px-2.5 py-2 text-left text-xs font-semibold">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="border-border/40 text-foreground border px-2.5 py-2 text-xs">{children}</td>
  ),
  blockquote: ({ children }) => (
    <blockquote className="border-border/60 text-muted-foreground my-2.5 border-l-4 py-1 pl-3 text-xs italic">
      {children}
    </blockquote>
  ),
  a: ({ children, href }) => (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="wrap-break-word cursor-pointer text-blue-500 underline-offset-1 hover:underline"
    >
      {children}
    </a>
  ),
  img: ({ src, alt }) => (
    <img src={src} alt={alt} className="border-border/40 my-2.5 h-auto max-w-full rounded border" />
  ),
  hr: () => <hr className="border-border/40 my-3 border-t" />,
};

interface MarkdownProps {
  children: string;
  className?: string;
}

export const Markdown: FC<MarkdownProps> = ({ children, className }) => (
  <div className={cn("text-foreground", className)}>
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
      {children.replace(/<!--[\s\S]*?-->/g, "")}
    </ReactMarkdown>
  </div>
);
