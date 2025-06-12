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
  <div className={cn("rounded-lg p-4", "shadow-md dark:shadow-lg")}>
    <div className="font-semibold mb-2">{role}</div>
    <div className="prose prose-sm dark:prose-invert max-w-none prose-headings:font-semibold prose-headings:tracking-tight prose-p:leading-7 prose-pre:bg-secondary prose-pre:p-0 prose-pre:rounded-md prose-pre:overflow-x-auto prose-pre:max-w-full prose-code:before:content-none prose-code:after:content-none">
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
  </div>
);

export const ErrorEvent = ({ content }: { content: string }) => (
  <div className="bg-destructive/10 text-destructive rounded-lg p-4 dark:bg-destructive/20 dark:text-destructive">
    <div className="font-semibold mb-2">⚠️ Error</div>
    <div>{content}</div>
  </div>
);

export const ChoiceEvents = ({ events }: { events: EventItem[] }) => {
  const content = events.map((e) => e.content).join("");
  const agent = events[0]?.metadata?.agent;

  return (
    <div className="bg-card text-card-foreground rounded-lg p-4 shadow-md dark:bg-card dark:text-card-foreground dark:shadow-lg">
      {agent && (
        <div className="font-semibold mb-2 dark:text-inherit">🤖 {agent}</div>
      )}
      <div className="prose prose-sm dark:prose-invert max-w-none prose-headings:font-semibold prose-headings:tracking-tight prose-p:leading-7 prose-pre:bg-secondary prose-pre:p-0 prose-pre:rounded-md prose-pre:overflow-x-auto prose-pre:max-w-full prose-code:before:content-none prose-code:after:content-none">
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
    </div>
  );
};
