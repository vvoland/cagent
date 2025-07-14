export interface ToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

export interface ChatCompletionMessage {
  role: string;
  content: string;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
}

export interface AgentMessage {
  agent: {
    name: string;
  };
  message: ChatCompletionMessage;
}

export interface ChatCompletionStreamChoice {
  delta: {
    content: string;
    role: string;
  };
}

export type EventType = "tool_call" | "tool_result" | "message" | "error";

export interface EventItem {
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

export interface Event {
  tool_call?: ToolCall;
  response?: string;
  message?: ChatCompletionMessage;
  choice?: ChatCompletionStreamChoice;
  agent?: string;
  error?: {
    message: string;
  };
}

export interface Agent {
  name: string;
  description: string;
  instruction: string;
  model: string;
}

export interface Session {
  id: string;
  created_at: string;
  messages: AgentMessage[];
  agents: { [key: string]: Agent };
}
