import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism";
import "./App.css";

interface ToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

interface ChatCompletionMessage {
  role: string;
  content: string;
  tool_calls?: ToolCall[];
}

interface ChatCompletionStreamChoice {
  delta: {
    content: string;
  };
}

type EventType = "choice" | "tool_call" | "tool_result" | "message" | "error";

interface EventItem {
  type: EventType;
  content: string;
  metadata?: {
    toolName?: string;
    toolArgs?: string;
    toolId?: string;
    role?: string;
    response?: string;
    agent?: string;
  };
}

interface Event {
  tool_call?: ToolCall;
  response?: string;
  message?: ChatCompletionMessage;
  choice?: ChatCompletionStreamChoice;
  agent?: string;
  error?: {
    message: string;
  };
}

// Modal component for displaying tool details
interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

const Modal = ({ isOpen, onClose, title, children }: ModalProps) => {
  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>{title}</h3>
          <button className="modal-close-btn" onClick={onClose}>
            ×
          </button>
        </div>
        <div className="modal-body">{children}</div>
      </div>
    </div>
  );
};

// Component for rendering tool calls
const ToolCallEvent = ({ name, args }: { name: string; args: string }) => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <div
        className="tool-pill tool-call-pill"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="tool-icon">🛠️</span>
        <span className="tool-name">{name}</span>
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Call: ${name}`}
      >
        <h4>Parameters:</h4>
        <pre className="tool-args">
          <code>{args}</code>
        </pre>
      </Modal>
    </>
  );
};

// Component for rendering tool results
const ToolResultEvent = ({ id, content }: { id: string; content: string }) => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <div
        className="tool-pill tool-result-pill"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="tool-icon">✅</span>
        <span className="tool-name">{id}</span>
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Result: ${id}`}
      >
        <h4>Response:</h4>
        <div
          className="tool-content"
          style={{ overflowY: "auto", maxHeight: "500px" }}
        >
          <pre>{content}</pre>
        </div>
      </Modal>
    </>
  );
};

// Component for rendering messages
const MessageEvent = ({ role, content }: { role: string; content: string }) => (
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

// Component for rendering errors
const ErrorEvent = ({ content }: { content: string }) => (
  <div className="error">
    <div className="error-header">⚠️ Error</div>
    <div className="error-content">{content}</div>
  </div>
);

// Component for rendering consecutive choice events as a single markdown
const ChoiceEvents = ({ events }: { events: EventItem[] }) => {
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

function App() {
  const [prompt, setPrompt] = useState("");
  const [events, setEvents] = useState<EventItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setEvents([]);

    try {
      const response = await fetch("/agent", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify([
          {
            role: "user",
            content: prompt,
          },
        ]),
      });

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error("No reader available");
      }

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        // Convert the Uint8Array to a string
        const text = new TextDecoder().decode(value);
        const lines = text.split("\n").filter((line) => line.trim());

        // Process each line
        lines.forEach((line) => {
          if (line.startsWith("data: ")) {
            try {
              const eventData = JSON.parse(line.slice(6)) as Event;
              console.log(eventData);

              // Handle different event types
              if (eventData.choice?.delta.content) {
                setEvents((prev) => {
                  const lastEvent = prev[prev.length - 1];
                  if (lastEvent?.type === "choice") {
                    // Append to the last choice event
                    return [
                      ...prev.slice(0, -1),
                      {
                        ...lastEvent,
                        content:
                          lastEvent.content + eventData.choice!.delta.content,
                      },
                    ];
                  }
                  // Create new choice event
                  return [
                    ...prev,
                    {
                      type: "choice",
                      content: eventData.choice!.delta.content,
                      metadata: {
                        agent: eventData.agent,
                      },
                    },
                  ];
                });
              } else if (eventData.tool_call?.function && !eventData.response) {
                const {
                  id,
                  function: { name, arguments: args },
                } = eventData.tool_call;
                setEvents((prev) => [
                  ...prev,
                  {
                    type: "tool_call",
                    content: "",
                    metadata: {
                      toolName: name,
                      toolArgs: args,
                      toolId: id,
                    },
                  },
                ]);
              } else if (eventData.response) {
                // Handle tool call result
                setEvents((prev) => {
                  const lastToolCall = [...prev]
                    .reverse()
                    .find((e) => e.type === "tool_call");
                  return [
                    ...prev,
                    {
                      type: "tool_result" as const,
                      content: eventData.response || "",
                      metadata: {
                        toolId: lastToolCall?.metadata?.toolName,
                      },
                    },
                  ];
                });
              } else if (eventData.message?.content) {
                setEvents((prev) => [
                  ...prev,
                  {
                    type: "message",
                    content: eventData.message!.content,
                    metadata: {
                      role: eventData.message!.role,
                    },
                  },
                ]);
              } else if (eventData.error?.message) {
                setEvents((prev) => [
                  ...prev,
                  {
                    type: "error",
                    content: eventData.error!.message,
                  },
                ]);
              }
            } catch (e) {
              console.error("Failed to parse event:", e);
            }
          }
        });
      }
    } catch (error) {
      console.error("Error:", error);
      setEvents((prev) => [
        ...prev,
        {
          type: "error",
          content: (error as Error).message,
        },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  // Group consecutive choice events together
  const groupedEvents = events.reduce<(EventItem | EventItem[])[]>(
    (acc, event) => {
      if (event.type === "choice") {
        const lastGroup = acc[acc.length - 1];
        if (Array.isArray(lastGroup) && lastGroup[0].type === "choice") {
          lastGroup.push(event);
        } else {
          acc.push([event]);
        }
      } else {
        acc.push(event);
      }
      return acc;
    },
    []
  );

  return (
    <div className="container">
      <div className="response">
        {groupedEvents.map((event, index) => {
          if (Array.isArray(event)) {
            return <ChoiceEvents key={index} events={event} />;
          }

          switch (event.type) {
            case "tool_call":
              return (
                <ToolCallEvent
                  key={index}
                  name={event.metadata?.toolName || ""}
                  args={event.metadata?.toolArgs || ""}
                />
              );
            case "tool_result":
              return (
                <ToolResultEvent
                  key={index}
                  id={event.metadata?.toolId || ""}
                  content={event.content}
                />
              );
            case "message":
              return (
                <MessageEvent
                  key={index}
                  role={event.metadata?.role || ""}
                  content={event.content}
                />
              );
            case "error":
              return <ErrorEvent key={index} content={event.content} />;
            default:
              return null;
          }
        })}
      </div>

      <form onSubmit={handleSubmit} className="form">
        <input
          type="text"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder="Enter your prompt..."
          disabled={isLoading}
          className="input"
        />
        <button type="submit" disabled={isLoading} className="button">
          {isLoading ? "Processing..." : "Submit"}
        </button>
      </form>
    </div>
  );
}

export default App;
