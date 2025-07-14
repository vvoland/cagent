import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism";
import { cn } from "../lib/utils";

export const MessageEvent = ({
  role,
  agent,
  content,
}: {
  role: string;
  agent: string | null;
  content: string;
}) => (
  <div
    className={cn("leading-8", "rounded-lg p-4", "shadow-md dark:shadow-lg")}
  >
    <div className="font-semibold mb-2">{agent ? `🤖 ${agent}` : role}</div>
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        code(props) {
          const { className, children } = props;
          const match = /language-(\w+)/.exec(className || "");
          return match ? (
            <SyntaxHighlighter
              style={vscDarkPlus}
              language={match[1]}
              PreTag="div"
              className="rounded-md !bg-secondary dark:!bg-secondary"
            >
              {String(children).replace(/\n$/, "")}
            </SyntaxHighlighter>
          ) : (
            <code
              className={cn(
                "bg-secondary px-1.5 py-0.5 rounded-md dark:bg-secondary dark:text-foreground",
                className
              )}
            >
              {children}
            </code>
          );
        },
      }}
    >
      {content}
    </ReactMarkdown>
  </div>
);

export const ErrorEvent = ({ content }: { content: string }) => (
  <div className="bg-destructive/10 text-destructive rounded-lg p-4 dark:bg-destructive/20 dark:text-destructive">
    <div className="font-semibold mb-2">⚠️ Error</div>
    <div>{content}</div>
  </div>
);
