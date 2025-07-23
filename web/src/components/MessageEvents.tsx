import { memo, useMemo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus, vs } from "react-syntax-highlighter/dist/esm/styles/prism";
import { cn } from "../lib/utils";
import { useDarkMode } from "../hooks/useDarkMode";
import { MessageActionButtons } from "./MessageActionButtons";

interface MessageEventProps {
  role: string;
  agent: string | null;
  content: string;
  onReplay?: (() => void) | undefined;
}

interface ErrorEventProps {
  content: string;
}

export const MessageEvent = memo<MessageEventProps>(({ role, agent, content, onReplay }) => {
  const { isDarkMode } = useDarkMode();
  
  // Memoize markdown components for better performance
  const markdownComponents = useMemo(() => ({
    code(props: any) {
      const { className, children } = props;
      const match = /language-(\w+)/.exec(className || "");
      return match ? (
        <SyntaxHighlighter
          style={isDarkMode ? vscDarkPlus : vs}
          language={match[1]}
          PreTag="div"
          className="rounded-md !bg-secondary dark:!bg-secondary overflow-x-auto"
          customStyle={{
            margin: 0,
            padding: '1rem',
            fontSize: '0.875rem',
            lineHeight: '1.25rem'
          }}
        >
          {String(children).replace(/\n$/, "")}
        </SyntaxHighlighter>
      ) : (
        <code
          className={cn(
            "bg-secondary px-1.5 py-0.5 rounded-md dark:bg-secondary dark:text-foreground font-mono text-sm",
            className
          )}
        >
          {children}
        </code>
      );
    },
    pre: ({ children, ...props }: any) => (
      <pre className="overflow-x-auto rounded-md bg-secondary p-4 text-sm" {...props}>
        {children}
      </pre>
    ),
    blockquote: ({ children, ...props }: any) => (
      <blockquote className="border-l-4 border-primary pl-4 italic text-muted-foreground" {...props}>
        {children}
      </blockquote>
    ),
    table: ({ children, ...props }: any) => (
      <div className="overflow-x-auto">
        <table className="min-w-full border-collapse" {...props}>
          {children}
        </table>
      </div>
    ),
    th: ({ children, ...props }: any) => (
      <th className="border border-border px-4 py-2 text-left font-semibold" {...props}>
        {children}
      </th>
    ),
    td: ({ children, ...props }: any) => (
      <td className="border border-border px-4 py-2" {...props}>
        {children}
      </td>
    ),
  }), [isDarkMode]);

  const displayName = agent ? `🤖 ${agent}` : role;

  return (
    <article
      className={cn(
        "group leading-8 rounded-lg p-3 lg:p-4 shadow-md dark:shadow-lg",
        "transition-all hover:shadow-lg dark:hover:shadow-xl",
        "border border-transparent hover:border-primary/20",
        "text-sm lg:text-base"
      )}
      role="article"
      aria-label={`Message from ${displayName}`}
    >
      <header className="font-semibold mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 flex-wrap min-w-0 flex-1">
          <span className="text-sm lg:text-base truncate">{displayName}</span>
          <span className="text-xs text-muted-foreground bg-secondary px-2 py-1 rounded-full flex-shrink-0">
            {role}
          </span>
        </div>
        
        {/* Message Action Buttons */}
        <MessageActionButtons
          content={content}
          role={role}
          onReplay={onReplay}
        />
      </header>
      
      <div className="prose prose-sm lg:prose-base max-w-none dark:prose-invert overflow-x-auto">
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={markdownComponents}
        >
          {content}
        </ReactMarkdown>
      </div>
    </article>
  );
});

MessageEvent.displayName = 'MessageEvent';

export const ErrorEvent = memo<ErrorEventProps>(({ content }) => (
  <div 
    className="bg-destructive/10 text-destructive rounded-lg p-3 lg:p-4 dark:bg-destructive/20 dark:text-destructive border border-destructive/20 transition-all hover:shadow-md"
    role="alert"
    aria-live="polite"
  >
    <div className="font-semibold mb-2 flex items-center gap-2">
      <span className="text-lg">⚠️</span>
      <span className="text-sm lg:text-base">Error</span>
    </div>
    <div className="font-mono text-xs lg:text-sm bg-destructive/5 p-2 rounded border border-destructive/10 overflow-x-auto">
      {content}
    </div>
  </div>
));

ErrorEvent.displayName = 'ErrorEvent';