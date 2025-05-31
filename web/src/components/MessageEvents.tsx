import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism";
import type { EventItem } from "../types";

export const MessageEvent = ({
  role,
  content,
}: {
  role: string;
  content: string;
}) => (
  <div className={`message ${role.toLowerCase()}`}>
    <div className="message-header">{role}</div>
    <div className="message-content">
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
              >
                {String(children).replace(/\n$/, "")}
              </SyntaxHighlighter>
            ) : (
              <code className={className}>{children}</code>
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
  <div className="error">
    <div className="error-header">⚠️ Error</div>
    <div className="error-content">{content}</div>
  </div>
);

export const ChoiceEvents = ({ events }: { events: EventItem[] }) => {
  const content = events.map((e) => e.content).join("");
  const agent = events[0]?.metadata?.agent;

  return (
    <div className="choice">
      {agent && <div className="agent-header">🤖 {agent}</div>}
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
              >
                {String(children).replace(/\n$/, "")}
              </SyntaxHighlighter>
            ) : (
              <code className={className}>{children}</code>
            );
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};
