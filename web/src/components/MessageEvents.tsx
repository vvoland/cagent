import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism";
import type { EventItem } from "../types";
import { cn } from "../lib/utils";

export const MessageEvent = ({
  role,
  content,
}: {
  role: string;
  content: string;
}) => (
  <div
    className={cn(
      "rounded-lg p-4",
      "shadow-md",
      role.toLowerCase() === "user"
        ? "bg-primary text-primary-foreground"
        : "bg-muted"
    )}
  >
    <div className="font-semibold mb-2">{role}</div>
    <div className="prose prose-sm dark:prose-invert max-w-none">
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
                className="rounded-md !bg-secondary"
              >
                {String(children).replace(/\n$/, "")}
              </SyntaxHighlighter>
            ) : (
              <code
                className={cn(
                  "bg-secondary px-1.5 py-0.5 rounded-md",
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
  </div>
);

export const ErrorEvent = ({ content }: { content: string }) => (
  <div className="bg-destructive/10 text-destructive rounded-lg p-4">
    <div className="font-semibold mb-2">⚠️ Error</div>
    <div>{content}</div>
  </div>
);

export const ChoiceEvents = ({ events }: { events: EventItem[] }) => {
  const content = events.map((e) => e.content).join("");
  const agent = events[0]?.metadata?.agent;

  return (
    <div className="bg-card text-card-foreground rounded-lg p-4 shadow-md">
      {agent && <div className="font-semibold mb-2">🤖 {agent}</div>}
      <div className="prose prose-sm dark:prose-invert max-w-none">
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
                  className="rounded-md !bg-secondary"
                >
                  {String(children).replace(/\n$/, "")}
                </SyntaxHighlighter>
              ) : (
                <code
                  className={cn(
                    "bg-secondary px-1.5 py-0.5 rounded-md",
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
    </div>
  );
};
